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
	"github.com/hex-techs/rocket/pkg/agent/workload/resourceoption"
	"github.com/hex-techs/rocket/pkg/util/condition"
	"github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	"github.com/hex-techs/rocket/pkg/util/tools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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
	kubeclientset kubernetes.Interface,
	kruiseclientset kclientset.Interface,
	rocketclientset clientset.Interface,
	workloadInformer informers.WorkloadInformer) *Controller {

	// Create event broadcaster
	// Add workload-controller types to the default Kubernetes Scheme so Events can be
	// logged for workload-controller types.
	utilruntime.Must(workloadscheme.AddToScheme(scheme.Scheme))

	controller := &Controller{
		kubeclientset:     kubeclientset,
		rocketclientset:   rocketclientset,
		kruiseclientset:   kruiseclientset,
		workloadsLister:   workloadInformer.Lister(),
		workloadsSynced:   workloadInformer.Informer().HasSynced,
		cronjobOption:     resourceoption.NewCronJobOption(kubeclientset),
		clonesetOption:    resourceoption.NewCloneSetOption(kruiseclientset),
		statefulsetOption: resourceoption.NewStatefulSetOption(kruiseclientset),
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Workloads"),
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	workloadInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWorkload,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueWorkload(new)
		},
		DeleteFunc: controller.enqueueWorkload,
	})

	return controller
}

// Controller is the controller implementation for Workload resources
type Controller struct {
	// 当前集群的 clientset 用来创建 cronjob 等工作负载
	kubeclientset kubernetes.Interface
	// 管理集群的 clientset，当发现管理端的 workload 资源变化时，触发本集群的修改
	rocketclientset clientset.Interface
	// 当前集群 open kruise 的 clientset
	kruiseclientset kclientset.Interface

	// manager 集群的 workload 资源
	workloadsLister listers.WorkloadLister
	workloadsSynced cache.InformerSynced

	cronjobOption     resourceoption.ResourceOption
	clonesetOption    resourceoption.ResourceOption
	statefulsetOption resourceoption.ResourceOption

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
		err = c.delete(workload)
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
			// 删除相关工作负载
			err = c.delete(workload)
			if err != nil {
				return err
			}
			workload.Finalizers = tools.RemoveString(workload.Finalizers, constant.WorkloadFinalizer)
			return c.updateWorkload(workload)
		}
	}
	// 处理deployment
	if workload.Spec.Template.DeploymentTemplate != nil {
		if err := c.handleDeployment(workload); err != nil {
			if len(workload.Status.Conditions) == 0 {
				workload.Status.Conditions = make(map[string]metav1.Condition)
			}
			workload.Status.Conditions[config.Pread().Name] = condition.GenerateCondition("Deployment", "Deployment", err.Error(), metav1.ConditionFalse)
		}
		workload.Status.Type = rocketv1alpha1.Stateless
	}
	// 处理cloneset
	if workload.Spec.Template.CloneSetTemplate != nil {
		if err := c.handleCloneSet(workload); err != nil {
			if len(workload.Status.Conditions) == 0 {
				workload.Status.Conditions = make(map[string]metav1.Condition)
			}
			workload.Status.Conditions[config.Pread().Name] = condition.GenerateCondition("CloneSet", "CloneSet", err.Error(), metav1.ConditionFalse)
		}
		workload.Status.Type = rocketv1alpha1.Stateless
	}
	// 处理cronjob
	if workload.Spec.Template.CronJobTemplate != nil {
		if err := c.handleCronjob(workload); err != nil {
			if len(workload.Status.Conditions) == 0 {
				workload.Status.Conditions = make(map[string]metav1.Condition)
			}
			workload.Status.Conditions[config.Pread().Name] = condition.GenerateCondition("CronJob", "CronJob", err.Error(), metav1.ConditionFalse)
		}
		workload.Status.Type = rocketv1alpha1.CronTask
	}
	// workload.Status.Phase = "Running"
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

func (c *Controller) delete(workload *rocketv1alpha1.Workload) error {
	if workload.Spec.Template.CloneSetTemplate != nil {
		name := tools.GenerateName(constant.Prefix, workload.Name)
		err := c.clonesetOption.Delete(name, workload.Namespace)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	if workload.Spec.Template.StatefulSetTemlate != nil {
		name := tools.GenerateName(constant.Prefix, workload.Name)
		err := c.statefulsetOption.Delete(name, workload.Namespace)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		// err = c.handleStsHeadlessSvc(workload, StsDeleteEvent)
		if err != nil {
			klog.V(4).Infof("failed to handle headless service of statefulset %v, err is %s", workload.Name, err)
		}
	}
	if workload.Spec.Template.CronJobTemplate != nil {
		name := tools.GenerateName(constant.Prefix, workload.Name)
		err := c.cronjobOption.Delete(name, workload.Namespace)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) handleDeployment(workload *rocketv1alpha1.Workload) error {
	// name := tools.GenerateName(constant.Prefix, workload.Name)
	// old := &appsv1.Deployment{}
	// deployment := &appsv1.Deployment{}
	// c.deploymentOption.Generate(name, workload, deployment)
	return nil
}

func (c *Controller) handleCloneSet(workload *rocketv1alpha1.Workload) error {
	name := tools.GenerateName(constant.Prefix, workload.Name)
	old := &kruiseappsv1alpha1.CloneSet{}
	cloneset := &kruiseappsv1alpha1.CloneSet{}
	c.clonesetOption.Generate(name, workload, cloneset)
	err := c.clonesetOption.Get(name, workload.Namespace, old)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		} else {
			err = c.clonesetOption.Create(name, cloneset.Namespace, *cloneset)
			if err != nil {
				return err
			}
		}
	} else {
		if !cmp.Equal(old.Spec, cloneset.Spec) {
			cloneset.ResourceVersion = old.ResourceVersion
			err = c.clonesetOption.Update(name, cloneset.Namespace, *cloneset)
			if err != nil {
				return err
			}
		}
	}
	exit := &kruiseappsv1alpha1.CloneSet{}
	err = c.clonesetOption.Get(name, cloneset.Namespace, exit)
	if err != nil {
		return err
	}
	workload.Status.ClonSetStatus = &exit.Status
	return nil
}

func (c *Controller) handleCronjob(workload *rocketv1alpha1.Workload) error {
	name := tools.GenerateName(constant.Prefix, workload.Name)
	old := &batchv1.CronJob{}
	cronjob := &batchv1.CronJob{}
	c.cronjobOption.Generate(name, workload, cronjob)
	err := c.cronjobOption.Get(name, workload.Namespace, old)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		} else {
			err = c.cronjobOption.Create(name, cronjob.Namespace, *cronjob)
			if err != nil {
				return err
			}
		}
	} else {
		if !cmp.Equal(old.Spec, cronjob.Spec) {
			cronjob.ResourceVersion = old.ResourceVersion
			err = c.cronjobOption.Update(name, cronjob.Namespace, *cronjob)
			if err != nil {
				return err
			}
		}
	}
	exit := &batchv1.CronJob{}
	err = c.cronjobOption.Get(name, cronjob.Namespace, exit)
	if err != nil {
		return err
	}
	workload.Status.CronjobStatus = &exit.Status
	workload.Status.Phase = "Running"
	return nil
}

// pod template 的兼容处理
func (c *Controller) compatiblePT(pt *v1.PodTemplateSpec) *v1.PodTemplateSpec {
	var container v1.Container
	length := len(pt.Spec.Containers)
	if length == 0 {
		return pt
	}
	container = pt.Spec.Containers[length-1]
	container.Resources.Limits = v1.ResourceList{
		v1.ResourceCPU:    *container.Resources.Limits.Cpu(),
		v1.ResourceMemory: *container.Resources.Limits.Memory(),
	}
	container.Resources.Requests = v1.ResourceList{
		v1.ResourceCPU:    *container.Resources.Requests.Cpu(),
		v1.ResourceMemory: *container.Resources.Requests.Memory(),
	}
	pt.Spec.Containers[length-1] = container
	return pt
}
