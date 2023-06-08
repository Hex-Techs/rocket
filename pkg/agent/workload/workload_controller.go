/*
Copyright2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// NOTE: Agent
package workload

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	workloadscheme "github.com/hex-techs/rocket/client/clientset/versioned/scheme"
	informers "github.com/hex-techs/rocket/client/informers/externalversions/rocket/v1alpha1"
	listers "github.com/hex-techs/rocket/client/listers/rocket/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/condition"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	StsCreateEvent = "create" // StsCreateEvent statefulset的创建事件
	StsDeleteEvent = "delete" // StsDeleteEvent statefulset的删除事件
	// SvcOpTypeCreate SvcOpTypeUpdate, SvcOpTypeDelete, SvcOpTypeSkip, SvcOpTypeAbort 是对statefulset的
	// headless service进行的操作
	SvcOpTypeCreate string = "create"
	SvcOpTypeUpdate string = "update"
	SvcOpTypeDelete string = "delete"
	SvcOpTypeSkip   string = "skip"
	SvcOpTypeAbort  string = "abort"
)

// NewController returns a new workload controller
func NewController(
	kubeclientset dynamic.Interface,
	rocketclientset clientset.Interface,
	workloadInformer informers.WorkloadInformer) *Controller {

	// Create event broadcaster
	// Add workload-controller types to the default Kubernetes Scheme so Events can be
	// logged for workload-controller types.
	utilruntime.Must(workloadscheme.AddToScheme(scheme.Scheme))

	controller := &Controller{
		kubeclientset:   kubeclientset,
		rocketclientset: rocketclientset,
		workloadsLister: workloadInformer.Lister(),
		workloadsSynced: workloadInformer.Informer().HasSynced,
		// extStatefulsetOption: resourceoption.NewExtStatefulSetOption(kruiseclientset),
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Workloads"),
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	workloadInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWorkload,
		UpdateFunc: func(old, new interface{}) {
			o, n := old.(*rocketv1alpha1.Workload).DeepCopy(), new.(*rocketv1alpha1.Workload).DeepCopy()
			needupdate := false
			needupdate = needupdate || !cmp.Equal(o.Spec, n.Spec)
			needupdate = needupdate || !cmp.Equal(o.ObjectMeta, n.ObjectMeta)
			oc, nc := map[string]metav1.Condition{}, map[string]metav1.Condition{}
			for k, v := range o.Status.Conditions {
				v.LastTransitionTime = metav1.Time{}
				oc[k] = v
			}
			for k, v := range n.Status.Conditions {
				v.LastTransitionTime = metav1.Time{}
				nc[k] = v
			}
			o.Status.Conditions, n.Status.Conditions = oc, nc
			if needupdate || !cmp.Equal(o.Status, n.Status) {
				controller.enqueueWorkload(new)
			}
		},
		DeleteFunc: controller.enqueueWorkload,
	})

	return controller
}

// Controller is the controller implementation for Workload resources
type Controller struct {
	// 当前集群的 clientset 用来创建 cronjob 等工作负载
	kubeclientset dynamic.Interface
	// 管理集群的 clientset，当发现管理端的 workload 资源变化时，触发本集群的修改
	rocketclientset clientset.Interface

	// manager 集群的 workload 资源
	workloadsLister listers.WorkloadLister
	workloadsSynced cache.InformerSynced

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
	klog.Info("Starting Workload controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.workloadsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workload workers")
	// Launch two workers to process Workload resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	klog.Info("Started workload workers")
	<-stopCh
	klog.Info("Shutting down workload workers")

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
		// Workload resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced workload '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Workload resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the Workload resource with this namespace/name
	workload, err := c.workloadsLister.Workloads(namespace).Get(name)
	if err != nil {
		// The Workload resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("workload '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	// 如果 workload 正在处于调度状态则不予理睬
	if workload.Status.Phase != "Running" {
		klog.V(3).Infof("got workload '%s' is in '%s' phase, skip it. Please use tools not edit it.", key, workload.Status.Phase)
		return nil
	}
	// 获取 workload 原有集群的信息
	oldClusterSet := mapset.NewSet[string]()
	if len(workload.Annotations) != 0 {
		if v, ok := workload.Annotations[constant.LastSchedulerClusterAnnotation]; ok {
			for _, c := range strings.Split(v, ",") {
				oldClusterSet.Add(c)
			}
		}
	}
	// 获取 workload 现有的集群信息
	set := mapset.NewSet[string]()
	currentClusters := []string{}
	if len(workload.Status.Clusters) == 0 {
		return nil
	} else {
		for _, c := range workload.Status.Clusters {
			set.Add(c)
			currentClusters = append(currentClusters, c)
		}
	}
	// 根据原有的和现有的集群信息判断如何处理
	// 1、当原有和现有集群信息都不包含本集群时，直接跳过不予处理
	// 2、当原有集群包含本集群，现有集群不包含，直接走删除逻辑，将本集群内的 cloneset 等负载删除，现有集群则创建
	// 3、其他情况则走正常的处理流程
	if !set.Contains(config.Pread().Name) && !oldClusterSet.Contains(config.Pread().Name) {
		return nil
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromWorkload(workload)
	if err != nil {
		return err
	}
	if resource == nil {
		return fmt.Errorf("resource template is nil")
	}
	gvr := gvktools.SetGVRForWorkload(gvk)
	if !set.Contains(config.Pread().Name) && oldClusterSet.Contains(config.Pread().Name) {
		// 原集群是本集群，新集群不是，处理删除工作负载的逻辑，要保证先创建，后删除
		for _, v := range currentClusters {
			if c, ok := workload.Status.Conditions[v]; ok {
				if c.Status != "True" {
					return fmt.Errorf("workload '%s/%s' has not been created in cluster '%s'. The workload corresponding to this cluster cannot be deleted until the creation is successful", namespace, name, v)
				}
			} else {
				return fmt.Errorf("workload '%s/%s' has not been created in cluster '%s'. The workload corresponding to this cluster cannot be deleted until the creation is successful", namespace, name, v)
			}
		}
		err = c.deleteWorkload(workload, gvr)
		if err != nil {
			return err
		}
		return nil
	}
	if workload.DeletionTimestamp.IsZero() {
		// 如果当前对象没有 finalizer， 说明其没有处于正被删除的状态。
		// 接着让我们添加 finalizer 并更新对象，相当于注册我们的 finalizer。
		if !tools.ContainsString(workload.ObjectMeta.Finalizers, constant.WorkloadFinalizer) {
			workload.ObjectMeta.Finalizers = append(workload.ObjectMeta.Finalizers, constant.WorkloadFinalizer)
			return c.updateWorkload(workload)
		}
	} else {
		if tools.ContainsString(workload.Finalizers, constant.WorkloadFinalizer) {
			if err := c.deleteWorkload(workload, gvr); err != nil {
				return err
			}
			workload.Finalizers = tools.RemoveString(workload.Finalizers, constant.WorkloadFinalizer)
			return c.updateWorkload(workload)
		}
	}
	// Convert the resource to JSON bytes
	b, _ := json.Marshal(resource)
	if err := c.createOrUpdateWorkload(workload, b, gvr); err != nil {
		if len(workload.Status.Conditions) == 0 {
			workload.Status.Conditions = make(map[string]metav1.Condition)
		}
		workload.Status.Conditions[config.Pread().Name] = condition.GenerateCondition(gvk.Kind, gvk.Kind, err.Error(), metav1.ConditionFalse)
	}
	switch gvr.Resource {
	case "deployments":
		workload.Status.Type = rocketv1alpha1.Stateless
	case "clonesets":
		workload.Status.Type = rocketv1alpha1.Stateless
	case "cronjobs":
		workload.Status.Type = rocketv1alpha1.CronTask
	}
	// workload status 由其他的控制器更新
	return c.updateWorkloadStatus(workload)
}

func (c *Controller) updateWorkload(workload *rocketv1alpha1.Workload) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	workloadCopy := workload.DeepCopy()
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Workload resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.rocketclientset.RocketV1alpha1().Workloads(workload.Namespace).Update(context.TODO(), workloadCopy, metav1.UpdateOptions{})
	return err
}

func (c *Controller) updateWorkloadStatus(workload *rocketv1alpha1.Workload) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	workloadCopy := workload.DeepCopy()
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Workload resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.rocketclientset.RocketV1alpha1().Workloads(workload.Namespace).UpdateStatus(context.TODO(), workloadCopy, metav1.UpdateOptions{})
	return err
}

// enqueueWorkload takes a Workload resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Workload.
func (c *Controller) enqueueWorkload(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// deleteWorkload deletes a workload
func (c *Controller) deleteWorkload(workload *rocketv1alpha1.Workload, gvr schema.GroupVersionResource) error {
	name := tools.GenerateName(constant.Prefix, workload.Name)
	return c.kubeclientset.Resource(gvr).Namespace(workload.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// createOrUpdateWorkload creates or updates a workload
func (c *Controller) createOrUpdateWorkload(workload *rocketv1alpha1.Workload, resbyte []byte, gvr schema.GroupVersionResource) error {
	name := tools.GenerateName(constant.Prefix, workload.Name)
	var resource *unstructured.Unstructured
	switch gvr.Resource {
	case "deployments":
		deploy := &appsv1.Deployment{}
		json.Unmarshal(resbyte, deploy)
		deployment := c.generateDeployment(name, workload, deploy)
		resource = gvktools.ConvertToUnstructured(deployment)
		klog.V(4).Infof("has resource 'deployments', generate resource: %v", resource)
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(resbyte, clone)
		cloneset := c.generateCloneSet(name, workload, clone)
		resource = gvktools.ConvertToUnstructured(cloneset)
		klog.V(4).Infof("has resource 'clonesets', generate resource: %v", resource)
	case "cronjobs":
		cj := &batchv1.CronJob{}
		json.Unmarshal(resbyte, cj)
		cronjob := c.generateCronjob(name, workload, cj)
		resource = gvktools.ConvertToUnstructured(cronjob)
		klog.V(4).Infof("has resource 'cronjobs', generate resource: %v", resource)
	default:
		klog.V(4).Infof("no resource '%s' found", gvr.Resource)
	}
	old, err := c.kubeclientset.Resource(gvr).Namespace(workload.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		} else {
			_, err := c.kubeclientset.Resource(gvr).Namespace(workload.Namespace).Create(context.TODO(), resource, metav1.CreateOptions{})
			return err
		}
	} else {
		if gvktools.NeedToUpdate(old, resource) {
			// 1. 设置labels
			newlables, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "labels")
			if err := unstructured.SetNestedField(old.Object, newlables, "metadata", "labels"); err != nil {
				return fmt.Errorf("set labels error: %v", err)
			}
			// 2. 设置annotations
			newannotations, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "annotations")
			if err := unstructured.SetNestedField(old.Object, newannotations, "metadata", "annotations"); err != nil {
				return fmt.Errorf("set annotations error: %v", err)
			}
			// 3. 设置spec
			newspec, found, _ := unstructured.NestedFieldCopy(resource.Object, "spec")
			if found {
				if err := unstructured.SetNestedField(old.Object, newspec, "spec"); err != nil {
					return fmt.Errorf("set spec error: %v", err)
				}
			}
			_, err := c.kubeclientset.Resource(gvr).Namespace(workload.Namespace).Update(context.TODO(), old, metav1.UpdateOptions{})
			return err
		}
	}
	exist, err := c.kubeclientset.Resource(gvr).Namespace(workload.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status, _, _ := unstructured.NestedFieldCopy(exist.Object, "status")
	workload.Status.WorkloadDetails = &runtime.RawExtension{}
	workload.Status.WorkloadDetails.Raw, _ = json.Marshal(status)
	return nil
}

func (c *Controller) generateDeployment(name string, workload *rocketv1alpha1.Workload, deploy *appsv1.Deployment) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
				constant.GenerateNameLabel: name,
				constant.AppNameLabel:      name,
			},
		},
		Spec: deploy.Spec,
	}
}

func (c *Controller) generateCloneSet(name string, workload *rocketv1alpha1.Workload, clone *kruiseappsv1alpha1.CloneSet) *kruiseappsv1alpha1.CloneSet {
	return &kruiseappsv1alpha1.CloneSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
				constant.GenerateNameLabel: name,
				constant.AppNameLabel:      name,
			},
		},
		Spec: clone.Spec,
	}
}

func (c *Controller) generateCronjob(name string, workload *rocketv1alpha1.Workload, cj *batchv1.CronJob) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
				constant.GenerateNameLabel: name,
				constant.AppNameLabel:      name,
			},
		},
		Spec: cj.Spec,
	}
}
