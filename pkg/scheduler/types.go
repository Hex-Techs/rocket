package scheduler

import (
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type ClusterClient struct {
	// 所在地域
	Region string
	// 所在云区域，专有云或者私有云
	Area string
	// 所在的环境
	Env string
	// 污点信息
	Taint           []v1.Taint
	Client          kubernetes.Interface
	DynamicClient   dynamic.Interface
	StopCh          <-chan struct{}
	Stop            chan struct{}
	CacheController *CacheController
}

func (c *ClusterClient) Controller() *CacheController {
	return c.CacheController
}

var (
	lock = &sync.Mutex{}
	// store the agent cluster client
	ClientMap sync.Map
)

type CacheController struct {
	name          string // name of the controller
	kubeclientset kubernetes.Interface
	clusterNodes  sync.Map
	nodesLister   corev1listers.NodeLister
	nodeSynced    cache.InformerSynced
	podLister     corev1listers.PodLister
	podSynced     cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
}

func NewCacheController(
	name string,
	kubeclientset kubernetes.Interface,
	nodeInformer corev1informers.NodeInformer,
	podInformer corev1informers.PodInformer) *CacheController {
	cachecontroller := &CacheController{
		name:          name,
		kubeclientset: kubeclientset,
		nodesLister:   nodeInformer.Lister(),
		nodeSynced:    nodeInformer.Informer().HasSynced,
		podLister:     podInformer.Lister(),
		podSynced:     podInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
	}
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: cachecontroller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			cachecontroller.enqueue(new)
		},
		DeleteFunc: cachecontroller.enqueue,
	})
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: cachecontroller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			cachecontroller.enqueue(new)
		},
		DeleteFunc: cachecontroller.enqueue,
	})
	return cachecontroller
}
func (c *CacheController) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
func (c *CacheController) Run(name string, workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	// Start the informer factories to begin populating the informer caches
	klog.Infof("Starting %s Cluster Controller", name)
	// Wait for the caches to be synced before starting workers
	klog.Infof("Waiting for %s cluster informer caches to sync", name)
	if ok := cache.WaitForCacheSync(stopCh, c.nodeSynced, c.podSynced); !ok {
		return fmt.Errorf("failed to wait for %s cluster caches to sync", name)
	}
	klog.Infof("Starting %s workers", name)
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	klog.Infof("Started %s workers", name)
	<-stopCh
	klog.Infof("Shutting down %s workers with stopCh", name)
	return nil
}
func (c *CacheController) runWorker() {
	for c.processNextWorkItem() {
	}
}
func (c *CacheController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}
func (c *CacheController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	if len(namespace) != 0 {
		pod, err := c.podLister.Pods(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !c.isNodeInfoExist(pod.Spec.NodeName) {
			klog.V(4).Infof("pod %s/%s is not on any node in cluster %s", pod.Namespace, pod.Name, c.name)
			return nil
		}
		if pod.DeletionTimestamp == nil {
			if v, ok := c.clusterNodes.Load(pod.Spec.NodeName); ok {
				nodeInfo := v.(*framework.NodeInfo)
				for idx, p := range nodeInfo.Pods {
					if p.Pod.Name == pod.Name {
						lock.Lock()
						// TODO: 此处会导致 OutOfIndex 异常
						nodeInfo.Pods[idx].Update(pod)
						c.clusterNodes.Store(pod.Spec.NodeName, nodeInfo)
						lock.Unlock()
						break
					}
					if idx == len(nodeInfo.Pods)-1 {
						lock.Lock()
						nodeInfo.AddPod(pod)
						c.clusterNodes.Store(pod.Spec.NodeName, nodeInfo)
						lock.Unlock()
					}
				}
			}
		} else {
			if v, ok := c.clusterNodes.Load(pod.Spec.NodeName); ok {
				nodeInfo := v.(*framework.NodeInfo)
				for _, p := range nodeInfo.Pods {
					if p.Pod.Name == pod.Name {
						lock.Lock()
						nodeInfo.RemovePod(pod)
						c.clusterNodes.Store(pod.Spec.NodeName, nodeInfo)
						lock.Unlock()
						break
					}
				}
				klog.V(4).Infof("pod %s/%s is deleted from node %s/%s", pod.Namespace, pod.Name, c.name, pod.Spec.NodeName)
			}
		}
	} else {
		node, err := c.nodesLister.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("node '%s' in work queue no longer exists in cluster %s", key, c.name))
				return nil
			}
			return err
		}
		if node.DeletionTimestamp != nil {
			if c.isNodeInfoExist(name) {
				c.clusterNodes.Delete(name)
			}
		} else {
			// 获取当前节点的所有pod
			var pods []*v1.Pod
			ret, err := c.podLister.Pods(v1.NamespaceAll).List(labels.Everything())
			if err != nil {
				return err
			}
			for _, pod := range ret {
				if pod.Spec.NodeName == node.Name {
					pods = append(pods, pod)
				}
			}
			n := framework.NewNodeInfo(pods...)
			n.SetNode(node)
			c.clusterNodes.Store(name, n)
		}
	}
	return nil
}
func (c *CacheController) isNodeInfoExist(name string) bool {
	_, ok := c.clusterNodes.Load(name)
	return ok
}
func (c *CacheController) List() []*framework.NodeInfo {
	var nodes []*framework.NodeInfo
	c.clusterNodes.Range(func(key, value interface{}) bool {
		n := value.(*framework.NodeInfo)
		nodes = append(nodes, n)
		return true
	})
	return nodes
}
