package cluster

import (
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	rocketclientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/util/clustertools"
	"github.com/hex-techs/rocket/pkg/util/config"
	agentconfig "github.com/hex-techs/rocket/pkg/util/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// 超时时间
	waitTime = 10 * time.Second

	clusterlabel = "rocket.hextech.io/cluster-name"
)

// 注册器，负责将集群注册到 manager
type register struct {
	// 当前集群的 client
	cli rocketclientset.Interface
	// 当前集群实例
	currentCluster *rocketv1alpha1.Cluster
	// 跳过节点的正则
	nodere *regexp.Regexp
	// 注册成功后，存储的 token
	// token string
	// 参数变量
	param *agentconfig.CommandParam
}

var (
	lock sync.RWMutex
	// 注册实例
	registerInstance *register
	// 心跳周期
	heartbeatTime time.Duration
	// 管理集群 client，当集群注册成功后会被替换成权限完成的 client
	// managerClient kubernetes.Interface

	// 集群状态
	State string
	// 该集群是否成功注册到 core
	registed bool = false
	// 是否创建 cluster
	create = true
)

func RegisterInit(param *agentconfig.CommandParam, mgr manager.Manager, p *agentconfig.ModuleParams) {
	heartbeatTime = time.Duration(param.KeepAliveSecond) * time.Second
	configfile, err := clustertools.GenerateKubeConfigFromToken(param.MasterURL, param.BootstrapToken, nil, 1)
	if err != nil {
		klog.Fatalf("generate config with error: %v", err)
	}
	registerInstance = &register{
		cli: rocketclientset.NewForConfigOrDie(configfile),
		currentCluster: &rocketv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: param.Name,
				Labels: map[string]string{
					clusterlabel: param.Name,
				},
			},
			Spec: rocketv1alpha1.ClusterSpec{
				Region:      param.Region,
				Area:        param.Area,
				Environment: param.Environment,
			},
		},
		param: param,
	}
	go func() {
		ticker := time.NewTicker(waitTime)
		defer ticker.Stop()
		for {
			_, ok := <-mgr.Elected()
			if !ok {
				break
			}
			time.Sleep(2 * time.Second)
		}
		for {
			klog.V(3).Infof("it is observed that the cluster is '%s'", State)
			if State == string(rocketv1alpha1.Approve) && registed {
				if err := installModulesInCluster(p, mgr, param.Name); err != nil {
					klog.Errorf("install modules with error: %v", err)
				}
				return
			}
			<-ticker.C
		}
	}()
}

func (r *register) isClusterExist() *register {
	cls, err := r.cli.RocketV1alpha1().Clusters().Get(context.TODO(),
		r.param.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r
		}
		klog.Fatalf("Determine whether the cluster exists with error: %v", err)
	}
	create = false
	State = string(cls.Status.State)
	r.currentCluster = cls
	return r
}

// register this cluster
func (r *register) registerCluster(mgr manager.Manager) *register {
	id := uuid.New().String()
	if create {
		r.currentCluster.Spec.ID = id
		_, err := handleControllerRevision(id, mgr)
		if err != nil {
			klog.Fatalf("register cluster failed with error: %v(create controllerrevision)", err)
		}
		ticker := time.NewTicker(waitTime)
		defer ticker.Stop()
		for {
			created, err := r.cli.RocketV1alpha1().Clusters().Create(context.TODO(),
				r.currentCluster, metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					<-ticker.C
					continue
				} else {
					klog.Fatalf("register cluster failed with error: %v(create)", err)
				}
			}
			lock.Lock()
			State = string(created.Status.State)
			r.currentCluster = created
			registed = true
			lock.Unlock()
			return r
		}
	} else {
		cr, err := handleControllerRevision(id, mgr)
		if err != nil {
			klog.Fatalf("register cluster failed with error: %v(exist)", err)
		}
		cid := &clusterID{}
		json.Unmarshal(cr.Data.Raw, cid)
		if r.currentCluster.Spec.ID != cid.ID {
			for {
				// 如果集群名称相同，但是 id 不同，则认为已有集群使用相同名字，当前 agent 夯住，通过重启更新
				klog.Errorf("name '%s' cluster already exist, but id is not match, please change name.", config.Pread().Name)
				klog.Infof("cr id :%s, cluster id: %s", cid.ID, r.currentCluster.Spec.ID)
				time.Sleep(waitTime)
			}
		} else {
			registed = true
		}
	}
	return r
}

func (r *register) isClusterApproveOrNot(mgr manager.Manager) *register {
	watcher, err := r.cli.RocketV1alpha1().Clusters().Watch(context.TODO(),
		metav1.ListOptions{
			LabelSelector: labels.FormatLabels(map[string]string{
				clusterlabel: config.Pread().Name,
			}),
		})
	if err != nil {
		klog.Errorf("watch cluster status with error %v", err)
		return r
	}
	defer watcher.Stop()
	for e := range watcher.ResultChan() {
		c, ok := e.Object.(*rocketv1alpha1.Cluster)
		if !ok {
			continue
		}
		if r.statusJudgement(&c.Status) {
			return r
		}
	}
	return r
}

func (r *register) syncAuthData(mgr manager.Manager) *register {
	cls, err := r.cli.RocketV1alpha1().Clusters().Get(context.TODO(), r.currentCluster.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("sync auth data get cluster with error: %v", err)
		return r
	}
	r.currentCluster = cls
	config := rest.CopyConfig(mgr.GetConfig())
	if r.currentCluster.Spec.APIServer == "" {
		r.currentCluster.Spec.APIServer = config.Host
	}
	if len(config.CAData) != 0 {
		r.currentCluster.Spec.CAData = config.CAData
	} else {
		r.currentCluster.Spec.CAData = r.getCAFromPublic(mgr)
	}
	r.currentCluster.Spec.CertData = config.CertData
	r.currentCluster.Spec.KeyData = config.KeyData
	r.currentCluster.Spec.Token = []byte(config.BearerToken)
	updated, err := r.cli.RocketV1alpha1().Clusters().Update(context.TODO(), r.currentCluster, metav1.UpdateOptions{})
	if err != nil {
		klog.V(0).Infof("sync auth data with error: %v", err)
		return r
	}
	r.currentCluster = updated
	return r
}

func (r *register) heartbeat(mgr manager.Manager) {
	klog.V(0).Info("start heartbeat goroutine")
	ticker := time.NewTicker(heartbeatTime)
	defer ticker.Stop()
	for range ticker.C {
		geted, err := r.cli.RocketV1alpha1().Clusters().Get(context.TODO(),
			r.currentCluster.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("heartbeat get cluster with error: %v", err)
			continue
		}
		lock.Lock()
		r.currentCluster = geted
		r.currentCluster.Status.LastKeepAliveTime = metav1.Now()
		lock.Unlock()
		klog.V(3).Infof("heartbeat current cluster '%s' success and status is %s",
			r.currentCluster.Name, r.currentCluster.Status.State)
		updated, err := r.cli.RocketV1alpha1().Clusters().UpdateStatus(context.TODO(),
			r.currentCluster, metav1.UpdateOptions{})
		if err != nil {
			klog.V(0).Infof("heartbeat failed with error: %v", err)
		} else {
			klog.V(3).Infof("heartbeat success")
			lock.Lock()
			r.currentCluster = updated
			lock.Unlock()
		}
	}
}

func (r *register) statusJudgement(status *rocketv1alpha1.ClusterStatus) bool {
	exit := false
	switch status.State {
	case rocketv1alpha1.Reject:
		klog.V(0).Info("cluster has been reject")
		State = string(rocketv1alpha1.Reject)
	case rocketv1alpha1.Approve:
		klog.V(0).Info("cluster was approve")
		State = string(rocketv1alpha1.Approve)
		exit = true
	case rocketv1alpha1.Offline:
		klog.V(0).Info("cluster was Offline, will send heartbeat later")
		State = string(rocketv1alpha1.Approve)
		exit = true
	}
	return exit
}

// 通过 kube-public 的 cluster-info 获取 ca
func (r *register) getCAFromPublic(mgr manager.Manager) []byte {
	klog.V(3).Info("get CertificateAuthorityData from 'kube-public/cluster-info'")
	cli, err := kubernetes.NewForConfig(rest.CopyConfig(mgr.GetConfig()))
	if err != nil {
		klog.Error(err)
		return []byte("")
	}
	cm, err := cli.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return []byte("")
	}
	ca := cm.Data["ca.crt"]
	return []byte(ca)
}
