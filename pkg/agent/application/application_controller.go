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

package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	appcheme "github.com/hex-techs/rocket/client/clientset/versioned/scheme"
	informers "github.com/hex-techs/rocket/client/informers/externalversions/rocket/v1alpha1"
	listers "github.com/hex-techs/rocket/client/listers/rocket/v1alpha1"
	"github.com/hex-techs/rocket/pkg/agent/application/trait"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	retryTime = 30 * time.Second
)

// NewController returns a new application controller
func NewController(
	kubeclientset kubernetes.Interface,
	dclientset dynamic.Interface,
	kruiseclientset kclientset.Interface,
	rocketclientset clientset.Interface,
	appinformers informers.ApplicationInformer) *Controller {

	// Create event broadcaster
	// Add application-controller types to the default Kubernetes Scheme so Events can be
	// logged for application-controller types.
	utilruntime.Must(appcheme.AddToScheme(scheme.Scheme))

	controller := &Controller{
		kubeclientset:    kubeclientset,
		dynamicclientset: dclientset,
		rocketclientset:  rocketclientset,
		kruiseclientset:  kruiseclientset,
		appLister:        appinformers.Lister(),
		appSynced:        appinformers.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "applications"),
	}

	log.Log.V(0).Info("Setting up event handlers")
	// Set up an event handler for when resources change
	appinformers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueApplication,
		UpdateFunc: func(old, new interface{}) {
			ot, nt := old.(*rocketv1alpha1.Application), new.(*rocketv1alpha1.Application)
			otCopy, ntCopy := ot.DeepCopy(), nt.DeepCopy()
			if !cmp.Equal(otCopy.GetGeneration(), ntCopy.GetGeneration()) ||
				!cmp.Equal(otCopy.GetDeletionTimestamp(), ntCopy.GetDeletionTimestamp()) ||
				!cmp.Equal(otCopy.Status.Clusters, ntCopy.Status.Clusters) {
				controller.enqueueApplication(new)
			}
		},
		DeleteFunc: controller.enqueueApplication,
	})
	return controller
}

// Controller is the controller implementation for application resources
type Controller struct {
	// 当前集群的 clientset
	kubeclientset kubernetes.Interface
	// 当前集群的 clientset
	dynamicclientset dynamic.Interface
	// 管理集群的 clientset，当发现管理端的 application 资源变化时，触发本集群的修改
	rocketclientset clientset.Interface
	// 当前集群 open kruise 的 clientset
	kruiseclientset kclientset.Interface

	appLister listers.ApplicationLister
	appSynced cache.InformerSynced

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
	log := log.FromContext(context.Background())
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	// Start the informer factories to begin populating the informer caches
	log.V(0).Info("Starting Application controller")
	// Wait for the caches to be synced before starting workers
	log.V(0).Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.appSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.V(0).Info("Starting application workers")
	// Launch two workers to process Workload resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	log.V(0).Info("Started application workers")
	<-stopCh
	log.V(0).Info("Shutting down application workers")
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
		// more up to date that when the item was initially put onto the·
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
		log.Log.V(0).Info("Successfully synced application", "application", key)
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
	log := log.FromContext(context.Background())
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the Application resource with this namespace/name
	application, err := c.appLister.Applications(namespace).Get(name)
	if err != nil {
		// The Application resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("application '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	// application is in current cluster or not
	if !c.isAppInCurrentCluster(application.Status.Clusters) {
		return nil
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromApplication(application)
	if err != nil {
		return err
	}
	if resource == nil {
		return fmt.Errorf("resource template is nil")
	}
	gvr := gvktools.SetGVRForApplication(gvk)
	if !application.DeletionTimestamp.IsZero() {
		// if application is deleting, delete all resources
		err = c.deleteTrait(application.Spec.Traits, application)
		c.applicationConditionsUpdate(application, metav1.ConditionTrue, constant.ReasonTraitDeleted, constant.MessageTraitDeleted)
		if err != nil {
			c.applicationConditionsUpdate(application, metav1.ConditionFalse, constant.ReasonTraitDeleted, err.Error())
			log.Error(err, "delete trait failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
		}
		err = c.deleteWorkload(application.Namespace, application.Name, gvr)
		c.applicationConditionsScale(application, metav1.ConditionTrue, c.getReason(gvr, false), constant.MessageApplicationDeleted)
		if err != nil {
			c.applicationConditionsScale(application, metav1.ConditionFalse, c.getReason(gvr, false), err.Error())
			log.Error(err, "delete resource failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
		}
		return c.updateApplicationStatus(application)
	}
	old, err := c.dynamicclientset.Resource(gvr).Namespace(application.Namespace).Get(context.TODO(), application.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(0).Info("resource is not found, create it", "workload", tools.KObj(resource), "application", tools.KObj(application))
			if gvr.Resource == "deployments" || gvr.Resource == "clonesets" {
				if err := unstructured.SetNestedField(resource.Object, *application.Spec.Replicas, "spec", "replicas"); err != nil {
					return fmt.Errorf("set replicas error: %v", err)
				}
			}
			_, err = c.dynamicclientset.Resource(gvr).Namespace(resource.GetNamespace()).Create(context.TODO(), resource, metav1.CreateOptions{})
			c.applicationConditionsScale(application, metav1.ConditionTrue, c.getReason(gvr, true), constant.MessageApplicationSynced)
			if err != nil {
				c.applicationConditionsScale(application, metav1.ConditionFalse, c.getReason(gvr, true), err.Error())
				log.Error(err, "create resource failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
			}
			err = c.handleTrait(application)
			c.applicationConditionsUpdate(application, metav1.ConditionTrue, constant.ReasonTraitSynced, constant.MessageTraitSynced)
			if err != nil {
				c.applicationConditionsUpdate(application, metav1.ConditionFalse, constant.ReasonTraitSynced, err.Error())
				log.Error(err, "handle trait failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
			}
			application.Status.Type = gvr.Resource
			return c.updateApplicationStatus(application)
		}
		return err
	}
	if gvktools.NeedToUpdate(old, resource) {
		if err := c.setWorkload(old, resource); err != nil {
			return err
		}
		if gvr.Resource == "deployments" || gvr.Resource == "clonesets" {
			if err := unstructured.SetNestedField(old.Object, *application.Spec.Replicas, "spec", "replicas"); err != nil {
				return fmt.Errorf("set replicas error: %v", err)
			}
		}
		_, err = c.dynamicclientset.Resource(gvr).Namespace(resource.GetNamespace()).Update(context.TODO(), old, metav1.UpdateOptions{})
		c.applicationConditionsScale(application, metav1.ConditionTrue, c.getReason(gvr, true), constant.MessageApplicationSynced)
		if err != nil {
			c.applicationConditionsScale(application, metav1.ConditionFalse, c.getReason(gvr, true), err.Error())
			log.Error(err, "update resource failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
		}
		err = c.handleTrait(application)
		c.applicationConditionsUpdate(application, metav1.ConditionTrue, constant.ReasonTraitSynced, constant.MessageTraitSynced)
		if err != nil {
			c.applicationConditionsUpdate(application, metav1.ConditionFalse, constant.ReasonTraitSynced, err.Error())
			log.Error(err, "handle trait failed", "workload", tools.KObj(resource), "application", tools.KObj(application))
		}
	}
	log.V(0).Info("origin data", "new application", resource)
	log.V(0).Info("origin data", "old application", old)
	return c.updateApplicationStatus(application)
}

func (c *Controller) isAppInCurrentCluster(clusters []string) bool {
	set := mapset.NewSet[string]()
	if len(clusters) == 0 {
		return false
	} else {
		for _, c := range clusters {
			set.Add(c)
		}
	}
	if !set.Contains(config.Pread().Name) {
		return false
	}
	return true
}

func (c *Controller) setWorkload(old, resource *unstructured.Unstructured) error {
	// 1. set labels
	newlables, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "labels")
	if err := unstructured.SetNestedField(old.Object, newlables, "metadata", "labels"); err != nil {
		return fmt.Errorf("set labels error: %v", err)
	}
	// 2. set spec
	newspec, _, _ := unstructured.NestedFieldCopy(resource.Object, "spec")
	if err := unstructured.SetNestedField(old.Object, newspec, "spec"); err != nil {
		return fmt.Errorf("set spec error: %v", err)
	}
	return nil
}

func (c *Controller) deleteWorkload(namespace, name string, res schema.GroupVersionResource) error {
	if err := c.dynamicclientset.Resource(res).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) getReason(res schema.GroupVersionResource, e bool) string {
	switch res.Resource {
	case "deployments":
		if e {
			return constant.DeploymentCreated
		}
		return constant.DeploymentDeleted
	case "clonesets":
		if e {
			return constant.CloneSetCreated
		}
		return constant.CloneSetDeleted
	case "cronjobs":
		if e {
			return constant.CronJobCreated
		}
		return constant.CronJobDeleted
	}
	return ""
}

func (c *Controller) applicationConditionsScale(application *rocketv1alpha1.Application, status metav1.ConditionStatus, reason, message string) {
	if len(application.Status.Conditions) == 0 {
		application.Status.Conditions = []rocketv1alpha1.ApplicationCondition{
			{
				Type:               rocketv1alpha1.ApplicationConditionSuccessedScale,
				Status:             status,
				LastTransitionTime: metav1.Now(),
				Reason:             reason,
				Message:            message,
			},
		}
		return
	}
	for i, c := range application.Status.Conditions {
		if c.Type == rocketv1alpha1.ApplicationConditionSuccessedScale {
			application.Status.Conditions[i].Status = status
			application.Status.Conditions[i].LastTransitionTime = metav1.Now()
			application.Status.Conditions[i].Reason = reason
			application.Status.Conditions[i].Message = message
			return
		}
		if i == len(application.Status.Conditions)-1 {
			application.Status.Conditions = append(application.Status.Conditions, rocketv1alpha1.ApplicationCondition{
				Type:               rocketv1alpha1.ApplicationConditionSuccessedScale,
				Status:             status,
				LastTransitionTime: metav1.Now(),
				Reason:             reason,
				Message:            message,
			})
			return
		}
	}
}

func (c *Controller) applicationConditionsUpdate(application *rocketv1alpha1.Application, status metav1.ConditionStatus, reason, message string) {
	if len(application.Status.Conditions) == 0 {
		application.Status.Conditions = []rocketv1alpha1.ApplicationCondition{
			{
				Type:               rocketv1alpha1.ApplicationConditionSuccessedUpdate,
				Status:             status,
				LastTransitionTime: metav1.Now(),
				Reason:             reason,
				Message:            message,
			},
		}
		return
	}
	for i, c := range application.Status.Conditions {
		if c.Type == rocketv1alpha1.ApplicationConditionSuccessedUpdate {
			application.Status.Conditions[i].Status = status
			application.Status.Conditions[i].LastTransitionTime = metav1.Now()
			application.Status.Conditions[i].Reason = reason
			application.Status.Conditions[i].Message = message
			return
		}
		if i == len(application.Status.Conditions)-1 {
			application.Status.Conditions = append(application.Status.Conditions, rocketv1alpha1.ApplicationCondition{
				Type:               rocketv1alpha1.ApplicationConditionSuccessedUpdate,
				Status:             status,
				LastTransitionTime: metav1.Now(),
				Reason:             reason,
				Message:            message,
			})
			return
		}
	}
}

func (c *Controller) updateApplicationStatus(application *rocketv1alpha1.Application) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	applicationCopy := application.DeepCopy()
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Workload resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.rocketclientset.RocketV1alpha1().Applications(application.Namespace).UpdateStatus(context.TODO(), applicationCopy, metav1.UpdateOptions{})
	return err
}

// enqueueApplication takes a Application resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Application.
func (c *Controller) enqueueApplication(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) getOldTrait(anno map[string]string) mapset.Set[string] {
	if anno == nil {
		return mapset.NewSet[string]()
	}
	if v, ok := anno[constant.TraitEdgeAnnotation]; !ok {
		return mapset.NewSet[string]()
	} else {
		set := mapset.NewSet[string]()
		for _, s := range strings.Split(v, ",") {
			set.Add(s)
		}
		return set
	}
}

func (c *Controller) getNewTrait(traits []rocketv1alpha1.Trait) mapset.Set[string] {
	set := mapset.NewSet[string]()
	for _, t := range traits {
		if _, ok := trait.Traits[t.Kind]; ok {
			set.Add(t.Kind)
		}
	}
	return set
}

func generateRemoveTrait(t []string) []rocketv1alpha1.Trait {
	traits := []rocketv1alpha1.Trait{}
	for _, v := range t {
		switch v {
		case trait.PodUnavailableBudgetKind:
			traits = append(traits, rocketv1alpha1.Trait{Kind: trait.PodUnavailableBudgetKind})
		}
	}
	return traits
}

func (c *Controller) handleTrait(app *rocketv1alpha1.Application) error {
	old := c.getOldTrait(app.Annotations)
	new := c.getNewTrait(app.Spec.Traits)
	removeStr := []string{}
	old.Difference(new).Each(func(s string) bool {
		removeStr = append(removeStr, s)
		return false
	})
	remove := generateRemoveTrait(removeStr)
	if err := c.deleteTrait(remove, app); err != nil {
		return err
	}
	addStr := []string{}
	new.Difference(old).Each(func(s string) bool {
		addStr = append(addStr, s)
		return false
	})
	return c.createOrUpdateTrait(app.Spec.Traits, app)
}

func (c *Controller) deleteTrait(ttmp []rocketv1alpha1.Trait, app *rocketv1alpha1.Application) error {
	for _, t := range ttmp {
		if opt, ok := trait.Traits[t.Kind]; ok {
			client := trait.NewClient(c.kruiseclientset, c.rocketclientset, c.kubeclientset)
			if err := opt.Handler(&t, app, trait.Deleted, client); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) createOrUpdateTrait(ttmp []rocketv1alpha1.Trait, app *rocketv1alpha1.Application) error {
	for _, t := range ttmp {
		if opt, ok := trait.Traits[t.Kind]; ok {
			client := trait.NewClient(c.kruiseclientset, c.rocketclientset, c.kubeclientset)
			if err := opt.Handler(&t, app, trait.CreatedOrUpdate, client); err != nil {
				return err
			}
		}
	}
	return nil
}
