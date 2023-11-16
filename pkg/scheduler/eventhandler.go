package scheduler

import (
	"fmt"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	schedcache "github.com/hex-techs/rocket/pkg/scheduler/cache"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	// Duration the scheduler will wait before expiring an assumed pod.
	durationToExpireAssumedPod = 15 * time.Minute
)

// the cache of the cluster
type clusterCache struct {
	// It is expected that changes made via Cache will be observed
	// by NodeLister and Algorithm.
	schedCache schedcache.Cache
	// region of the cluster
	region string
	// cloudarea pub or ded
	area string
	name string
	// cluster taint
	Taint []v1.Taint

	client kubernetes.Interface

	nodesLister corev1listers.NodeLister
	nodeSynced  cache.InformerSynced

	podLister corev1listers.PodLister
	podSynced cache.InformerSynced
}

func (cc *clusterCache) addPodToCache(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		log.Error(nil, "Cannot convert to *v1.Pod", "obj", obj)
		return
	}
	log.Debugw("Add event for scheduled pod", "pod", tools.KObj(pod))
	if err := cc.schedCache.AddPod(pod); err != nil {
		log.Error(err, "Scheduler cache AddPod failed", "pod", tools.KObj(pod))
	}
}

func (sched *clusterCache) updatePodInCache(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		log.Error(nil, "Cannot convert oldObj to *v1.Pod", "oldObj", oldObj)
		return
	}
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		log.Error(nil, "Cannot convert newObj to *v1.Pod", "newObj", newObj)
		return
	}
	log.Debugw("Update event for scheduled pod", "pod", tools.KObj(oldPod))
	if err := sched.schedCache.UpdatePod(oldPod, newPod); err != nil {
		log.Error(err, "Scheduler cache UpdatePod failed", "pod", tools.KObj(oldPod))
	}
}

func (sched *clusterCache) deletePodFromCache(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			log.Error(nil, "Cannot convert to *v1.Pod", "obj", t.Obj)
			return
		}
	default:
		log.Error(nil, "Cannot convert to *v1.Pod", "obj", t)
		return
	}
	log.Debugw("Delete event for scheduled pod", "pod", tools.KObj(pod))
	if err := sched.schedCache.RemovePod(pod); err != nil {
		log.Error(err, "Scheduler cache RemovePod failed", "pod", tools.KObj(pod))
	}
}

func (sched *clusterCache) addNodeToCache(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		log.Error(nil, "Cannot convert to *v1.Node", "obj", obj)
		return
	}

	sched.schedCache.AddNode(node)
	log.Debugw("Add event for node", "node", tools.KObj(node))
}

func (sched *clusterCache) updateNodeInCache(oldObj, newObj interface{}) {
	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		log.Error(nil, "Cannot convert oldObj to *v1.Node", "oldObj", oldObj)
		return
	}
	newNode, ok := newObj.(*v1.Node)
	if !ok {
		log.Error(nil, "Cannot convert newObj to *v1.Node", "newObj", newObj)
		return
	}
	sched.schedCache.UpdateNode(oldNode, newNode)
}

func (sched *clusterCache) deleteNodeFromCache(obj interface{}) {
	var node *v1.Node
	switch t := obj.(type) {
	case *v1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*v1.Node)
		if !ok {
			log.Error(nil, "Cannot convert to *v1.Node", "obj", t.Obj)
			return
		}
	default:
		log.Error(nil, "Cannot convert to *v1.Node", "obj", t)
		return
	}
	log.Debugw("Delete event for node", "node", tools.KObj(node))
	if err := sched.schedCache.RemoveNode(node); err != nil {
		log.Error(err, "Scheduler cache RemoveNode failed")
	}
}

// NewClusterCache return a new clusterCache。
func NewClusterCache(
	cls *cluster.Cluster,
	kubeclientset kubernetes.Interface,
	nodeInformer corev1informers.NodeInformer,
	podInformer corev1informers.PodInformer) *clusterCache {
	stop := make(chan struct{})
	clusterCacheController := &clusterCache{
		name:        cls.Name,
		region:      cls.Region,
		area:        cls.CloudArea,
		schedCache:  schedcache.New(durationToExpireAssumedPod, stop),
		client:      kubeclientset,
		nodesLister: nodeInformer.Lister(),
		nodeSynced:  nodeInformer.Informer().HasSynced,
		podLister:   podInformer.Lister(),
		podSynced:   podInformer.Informer().HasSynced,
	}
	podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				return assignedPod(t)
			case cache.DeletedFinalStateUnknown:
				if _, ok := t.Obj.(*v1.Pod); ok {
					// The carried object may be stale, so we don't use it to check if
					// it's assigned or not. Attempting to cleanup anyways.
					return true
				}
				utilruntime.HandleError(fmt.Errorf("unable to convert object %T to *v1.Pod in %T", obj, clusterCacheController))
				return false
			default:
				utilruntime.HandleError(fmt.Errorf("unable to handle object in %T: %T", clusterCacheController, obj))
				return false
			}
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    clusterCacheController.addPodToCache,
			UpdateFunc: clusterCacheController.updatePodInCache,
			DeleteFunc: clusterCacheController.deletePodFromCache,
		},
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    clusterCacheController.addNodeToCache,
		UpdateFunc: clusterCacheController.updateNodeInCache,
		DeleteFunc: clusterCacheController.deleteNodeFromCache,
	})

	return clusterCacheController
}

// assignedPod selects pods that are assigned (scheduled and running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}
