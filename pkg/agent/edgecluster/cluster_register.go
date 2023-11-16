package edgecluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	waitTime = 10 * time.Second
)

// 注册器，负责将集群注册到 manager
type register struct {
	// 当前集群的 client
	cli client.Interface
	// 当前集群实例
	currentCluster *cluster.Cluster
	// 参数变量
	param *config.AgentServiceConfig
}

var (
	lock sync.RWMutex
	// 注册实例
	registerClusterInstance *register
	// 心跳周期
	heartbeatTime time.Duration
	// 集群状态
	State string
	// 是否创建 cluster
	create = true
)

func RegisterInit(param *config.AgentServiceConfig) {
	heartbeatTime = time.Duration(param.KeepAliveSecond) * time.Second
	log.Infow("agent register cluster information", "MasterURL", param.ManagerAddress, "Token", param.BootstrapToken)
	registerClusterInstance = &register{
		cli: client.New(param.ManagerAddress, param.BootstrapToken),
		currentCluster: &cluster.Cluster{
			Name:      param.Name,
			Region:    param.Region,
			CloudArea: param.Area,
			Agent:     true,
		},
		param: param,
	}
}

func (r *register) isClusterExist() *register {
	var cls cluster.Cluster
	ticker := time.NewTicker(waitTime)
	defer ticker.Stop()
	for {
		err := r.cli.Get(r.param.Name, nil, &cls)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return r
			}
			log.Errorf("determine whether the cluster exists: %v", err)
			<-ticker.C
			continue
		}
		create = false
		State = string(cls.State)
		r.currentCluster = &cls
		return r
	}
}

// register this cluster
func (r *register) registerCluster() *register {
	if create {
		ticker := time.NewTicker(waitTime)
		defer ticker.Stop()
		for {
			err := r.cli.Create(r.currentCluster)
			if err != nil {
				log.Errorf("register cluster failed when create: %v", err)
				<-ticker.C
				continue
			}
			var created cluster.Cluster
			r.cli.Get(r.currentCluster.Name, nil, &created)
			lock.Lock()
			State = string(created.State)
			r.currentCluster = &created
			lock.Unlock()
			return r
		}
	} else {
	}
	return r
}

func (r *register) isClusterApproveOrNot() *register {
	messages := communication.GetChan()
	for msg := range messages {
		if m, ok := msg.(cache.Message); ok {
			if m.Kind == "cluster" && m.ActionType == cache.UpdateAction && m.Name == r.param.Name {
				var cls cluster.Cluster
				if err := r.cli.Get(m.Name, nil, &cls); err != nil {
					log.Errorf("get cluster: %v", err)
					continue
				}
				if r.statusJudgement(&cls) {
					return r
				}
			}
		}
	}
	return r
}

func (r *register) syncAuthData(mgr manager.Manager) *register {
	var cls cluster.Cluster
	if err := r.cli.Get(r.param.Name, nil, &cls); err != nil {
		log.Errorf("sync auth data get cluster: %v", err)
		return r
	}
	r.currentCluster = &cls
	config := rest.CopyConfig(mgr.GetConfig())
	if r.currentCluster.APIServer == "" {
		r.currentCluster.APIServer = config.Host
	}
	if len(config.CAData) != 0 {
		r.currentCluster.CAData = string(config.CAData)
	} else {
		r.currentCluster.CAData = string(r.getCAFromPublic(mgr))
	}
	r.currentCluster.CertData = string(config.CertData)
	r.currentCluster.KeyData = string(config.KeyData)
	r.currentCluster.Token = config.BearerToken
	if err := r.cli.Update(fmt.Sprintf("%d", r.currentCluster.ID), r.currentCluster); err != nil {
		log.Errorf("sync auth data: %v", err)
		return r
	}
	return r
}

func (r *register) heartbeat() {
	log.Info("start heartbeat goroutine")
	ticker := time.NewTicker(heartbeatTime)
	defer ticker.Stop()
	for range ticker.C {
		log.Infow("heartbeat", "Cluster", r.currentCluster.Name, "Status", r.currentCluster.State)
		if err := communication.SendMsg(&cache.Message{
			Name:        r.currentCluster.Name,
			Kind:        "cluster",
			MessageType: cache.Heartbeat,
		}); err != nil {
			log.Errorw("send heartbeat message error", "error", err)
		}
	}
}

func (r *register) statusJudgement(cls *cluster.Cluster) bool {
	exit := false
	switch cls.State {
	case cluster.Reject:
		log.Info("cluster has been reject")
		State = string(cluster.Reject)
	case cluster.Approve:
		log.Info("cluster was approve")
		State = string(cluster.Approve)
		exit = true
	case cluster.Offline:
		log.Info("cluster was Offline, will send heartbeat later")
		State = string(cluster.Approve)
		exit = true
	}
	return exit
}

// 通过 kube-public 的 cluster-info 获取 ca
func (r *register) getCAFromPublic(mgr manager.Manager) []byte {
	log.Debug("get CertificateAuthorityData from 'kube-public/cluster-info'")
	cli, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		log.Error(err, "generate client failed when get ca from 'kube-public/cluster-info'")
		return []byte("")
	}
	cm, err := cli.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		log.Error(err, "get configmap 'kube-root-ca.crt' failed")
		return []byte("")
	}
	ca := cm.Data["ca.crt"]
	return []byte(ca)
}
