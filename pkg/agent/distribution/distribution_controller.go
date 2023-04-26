// TODO: 存在一个问题，如果修改集群后立即删除，可能会因为控制器处理缓慢而这时又停止了控制器导致资源未被删除
package distribution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	rdsscheme "github.com/hex-techs/rocket/client/clientset/versioned/scheme"
	informers "github.com/hex-techs/rocket/client/informers/externalversions/rocket/v1alpha1"
	listers "github.com/hex-techs/rocket/client/listers/rocket/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	// An unknown problem has occurred
	UnknowReason = "Unknow"

	// The resource is ready or not
	ResourceReadyReason = "ResourceReady"

	// The resource is deleted
	ResourceDeleteReason = "ResourceDeleted"

	// The resource is ready
	ResourceReadyMessage = "Resource is ready"

	// The resource is deleted
	ResourceDeleteMessage = "Resource is deleted"
)

// NewController returns a new trait controller
func NewController(
	kubeclientset dynamic.Interface,
	discoveryClient *discovery.DiscoveryClient,
	rocketclientset clientset.Interface,
	rdsinformers informers.DistributionInformer) *Controller {

	// Create event broadcaster
	utilruntime.Must(rdsscheme.AddToScheme(scheme.Scheme))

	controller := &Controller{
		kubeclientset:   kubeclientset,
		discoveryClient: discoveryClient,
		rocketclientset: rocketclientset,
		rdsLister:       rdsinformers.Lister(),
		rdsSynced:       rdsinformers.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "distributions"),
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when resources change
	rdsinformers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuedistribution,
		UpdateFunc: func(old, new interface{}) {
			oldObj := old.(*rocketv1alpha1.Distribution)
			newObj := new.(*rocketv1alpha1.Distribution)
			if !newObj.DeletionTimestamp.IsZero() || !cmp.Equal(oldObj.Spec, newObj.Spec) {
				controller.enqueuedistribution(newObj)
			}
		},
		DeleteFunc: controller.enqueuedistribution,
	})

	return controller
}

// Controller is the controller implementation for distribution resources
type Controller struct {
	// 当前集群的 clientset 用来创建资源
	kubeclientset   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	// 管理集群的 clientset，当发现管理端的 distribution 资源变化时，触发本集群的修改
	rocketclientset clientset.Interface

	// manager 集群的 distribution 资源
	rdsLister listers.DistributionLister
	rdsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	// recorder record.EventRecorder
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting distribution controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.rdsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting distribution workers")
	// Launch two workers to process distribution resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	klog.Info("Started distribution workers")
	<-stopCh
	klog.Info("Shutting down distribution workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// distribution resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced distribution '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the distribution resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the distribution resource with this namespace/name
	rd, err := c.rdsLister.Distributions(namespace).Get(name)
	if err != nil {
		// The distribution resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("distribution '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	if len(rd.Spec.Resource.Raw) == 0 {
		return nil
	}
	resource, gvk, m, err := gvktools.Match(rd)
	if err != nil {
		return err
	}
	if resource == nil {
		// NOTE: 如果没有匹配到资源，那么就不需要处理了
		return nil
	}
	res := gvktools.SetGVRForDistribution(gvk)
	namespaced, err := gvktools.NamespacedScope(resource, c.discoveryClient)
	if err != nil {
		rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, UnknowReason, err.Error())
		return c.updatedistributionStatus(rd)
	}
	oldCond := c.oldCondition(rd.Status.Conditions)
	if !rd.DeletionTimestamp.IsZero() {
		if tools.ContainsString(rd.Finalizers, constant.DistributionFinalizer) {
			if namespaced {
				err = c.kubeclientset.Resource(res).Namespace(resource.GetNamespace()).Delete(context.TODO(), resource.GetName(), metav1.DeleteOptions{})
			} else {
				if gvk.Kind != "Namespace" {
					err = c.kubeclientset.Resource(res).Delete(context.TODO(), resource.GetName(), metav1.DeleteOptions{})
				}
			}
			if m {
				if err != nil && !errors.IsNotFound(err) {
					rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, ResourceDeleteReason, err.Error())
				} else {
					rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionTrue, ResourceDeleteReason, ResourceDeleteMessage)
				}
				klog.V(4).Infof("delete resource %s/%s is match current cluster", resource.GetNamespace(), resource.GetName())
				n := rd.Status.Conditions[config.Pread().Name]
				if !c.equalCondition(oldCond, &n) {
					return c.updatedistributionStatus(rd)
				}
				return nil
			}
		}
	}
	oldResource := &unstructured.Unstructured{}
	if namespaced {
		oldResource, err = c.kubeclientset.Resource(res).Namespace(resource.GetNamespace()).Get(context.TODO(), resource.GetName(), metav1.GetOptions{})
	} else {
		oldResource, err = c.kubeclientset.Resource(res).Get(context.TODO(), resource.GetName(), metav1.GetOptions{})
	}
	if err != nil {
		if errors.IsNotFound(err) {
			if !m {
				// 资源未匹配到本集群，无需处理
				return nil
			}
			// NOTE: 如果资源不存在，那么就创建
			if namespaced {
				_, err = c.kubeclientset.Resource(res).Namespace(resource.GetNamespace()).Create(context.TODO(), resource, metav1.CreateOptions{})
			} else {
				_, err = c.kubeclientset.Resource(res).Create(context.TODO(), resource, metav1.CreateOptions{})
			}
			if err != nil {
				rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, ResourceReadyReason, err.Error())
			} else {
				rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionTrue, ResourceReadyReason, ResourceReadyMessage)
			}
			return c.updatedistributionStatus(rd)
		}
		rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, ResourceReadyReason, err.Error())
		return c.updatedistributionStatus(rd)
	} else {
		if !m {
			// 资源存在却没有匹配到当前集群，则可能是之前匹配到过，删除即可
			if _, ok := oldResource.GetLabels()[constant.ManagedByRocketLabel]; ok {
				if namespaced {
					err = c.kubeclientset.Resource(res).Namespace(resource.GetNamespace()).Delete(context.TODO(), resource.GetName(), metav1.DeleteOptions{})
				} else {
					if gvk.Kind != "Namespace" {
						err = c.kubeclientset.Resource(res).Delete(context.TODO(), resource.GetName(), metav1.DeleteOptions{})
					}
				}
				if err != nil && !errors.IsNotFound(err) {
					rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, ResourceDeleteReason, err.Error())
				} else {
					rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionTrue, ResourceDeleteReason, ResourceDeleteMessage)
				}
				klog.V(4).Infof("delete resource %s/%s is not match current cluster", resource.GetNamespace(), resource.GetName())
				n := rd.Status.Conditions[config.Pread().Name]
				if !c.equalCondition(oldCond, &n) {
					return c.updatedistributionStatus(rd)
				}
				return nil
			}
		}
	}
	if gvktools.NeedToUpdate(oldResource, resource) && m {
		// NOTE: 如果资源存在，但是需要更新，那么就更新
		// 1. 设置labels
		newlables, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "labels")
		if err := unstructured.SetNestedField(oldResource.Object, newlables, "metadata", "labels"); err != nil {
			return fmt.Errorf("set labels error: %v", err)
		}
		// 2. 设置annotations
		newannotations, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "annotations")
		if err := unstructured.SetNestedField(oldResource.Object, newannotations, "metadata", "annotations"); err != nil {
			return fmt.Errorf("set annotations error: %v", err)
		}
		// 3. 设置spec
		newspec, found, _ := unstructured.NestedFieldCopy(resource.Object, "spec")
		if found {
			if err := unstructured.SetNestedField(oldResource.Object, newspec, "spec"); err != nil {
				return fmt.Errorf("set spec error: %v", err)
			}
		} else {
			// 4. 设置 data
			newdata, found, _ := unstructured.NestedFieldCopy(resource.Object, "data")
			if !found {
				return fmt.Errorf("not found spec or data")
			}
			if err := unstructured.SetNestedField(oldResource.Object, newdata, "data"); err != nil {
				return fmt.Errorf("set data error: %v", err)
			}
		}
		if namespaced {
			_, err = c.kubeclientset.Resource(res).Namespace(resource.GetNamespace()).Update(context.TODO(), oldResource, metav1.UpdateOptions{})
		} else {
			_, err = c.kubeclientset.Resource(res).Update(context.TODO(), oldResource, metav1.UpdateOptions{})
		}
		if err != nil {
			rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionFalse, ResourceReadyReason, err.Error())
		} else {
			rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionTrue, ResourceReadyReason, ResourceReadyMessage)
		}
		return c.updatedistributionStatus(rd)
	}
	rd.Status.Conditions = c.condition(rd.Status.Conditions, metav1.ConditionTrue, ResourceReadyReason, ResourceReadyMessage)
	return c.updatedistributionStatus(rd)
}

func (c *Controller) updatedistributionStatus(distribution *rocketv1alpha1.Distribution) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	distributionCopy := distribution.DeepCopy()
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the distribution resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.rocketclientset.RocketV1alpha1().Distributions(distribution.Namespace).UpdateStatus(context.TODO(), distributionCopy, metav1.UpdateOptions{})
	return err
}

// enqueuedistribution takes a distribution resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than distribution.
func (c *Controller) enqueuedistribution(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) condition(condition map[string]rocketv1alpha1.DistributionCondition,
	s metav1.ConditionStatus, r, m string) map[string]rocketv1alpha1.DistributionCondition {
	if condition == nil {
		condition = map[string]rocketv1alpha1.DistributionCondition{}
	}
	condition[config.Pread().Name] = rocketv1alpha1.DistributionCondition{
		LastTransitionTime: metav1.Now(),
		Status:             s,
		Reason:             r,
		Message:            m,
	}
	return condition
}

func (c *Controller) equalCondition(old, new *rocketv1alpha1.DistributionCondition) bool {
	if old == nil {
		return false
	}
	old.LastTransitionTime = metav1.Time{}
	new.LastTransitionTime = metav1.Time{}
	klog.V(3).Infof("old condition is %v, new condition is %v", old, new)
	return cmp.Equal(old, new)
}

func (c *Controller) oldCondition(o map[string]rocketv1alpha1.DistributionCondition) *rocketv1alpha1.DistributionCondition {
	if len(o) == 0 {
		return nil
	}
	if v, ok := o[config.Pread().Name]; ok {
		return &v
	}
	return nil
}
