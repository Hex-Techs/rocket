package localcache

import (
	"sync"

	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/signals"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Clients map[uint]clusterCache
	lock    sync.Mutex
)

func GetClient(id uint) (*clusterCache, bool) {
	lock.Lock()
	defer lock.Unlock()
	if c, ok := Clients[id]; ok {
		return &c, true
	}
	return nil, false
}

func AddClient(c *clusterCache) {
	lock.Lock()
	defer lock.Unlock()
	Clients[c.ID] = *c
}

func DeleteClient(id uint) {
	lock.Lock()
	defer lock.Unlock()
	if c, ok := Clients[id]; ok {
		c.Close()
	}
	delete(Clients, id)
}

func NewClusterCache(c *cluster.Cluster) (*clusterCache, error) {
	if c.Kubeconfig != "" {
		clientcmd.Load([]byte(c.Kubeconfig))
	}
	cfg, err := clustertools.GenerateRestConfigFromCluster(c)
	if err != nil {
		return nil, err
	}
	kubecli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	kruisecli, err := kruiseclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cls := &clusterCache{
		ID:        c.ID,
		Region:    c.Region,
		Area:      c.CloudArea,
		Name:      c.Name,
		config:    cfg,
		kubecli:   kubecli,
		kruisecli: kruisecli,
		stop:      make(chan struct{}),
	}
	cls.stopCh = signals.SetupSignalHandler(cls.stop)
	return cls, nil
}

type clusterCache struct {
	ID        uint
	Region    string
	Area      string
	Name      string
	config    *rest.Config
	kubecli   kubernetes.Interface
	kruisecli kruiseclientset.Interface
	stopCh    <-chan struct{}
	stop      chan struct{}
}

func (c *clusterCache) Close() {
	close(c.stop)
}

func (c *clusterCache) GetKubeConfig() *rest.Config {
	return c.config
}
