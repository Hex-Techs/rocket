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
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/client"
	rocketcli "github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	"github.com/hex-techs/rocket/pkg/utils/web"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"golang.org/x/time/rate"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

const (
	retryTime = 30 * time.Second
)

// NewReconciler returns a new application controller
func NewReconciler(
	kubeclientset kubernetes.Interface,
	dclientset dynamic.Interface,
	kruiseclientset kclientset.Interface) *ApplicationReconciler {
	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)
	controller := &ApplicationReconciler{
		kubeclientset:    kubeclientset,
		dynamicclientset: dclientset,
		kruiseclientset:  kruiseclientset,
		cli:              client.New(config.Read().Manager.InternalAddress, config.Read().Manager.AgentBootstrapToken),
		queue:            workqueue.NewRateLimitingQueue(ratelimiter),
	}
	return controller
}

type ApplicationReconciler struct {
	// 当前集群的 clientset
	kubeclientset kubernetes.Interface
	// 当前集群的 clientset
	dynamicclientset dynamic.Interface
	// 当前集群 open kruise 的 clientset
	kruiseclientset kclientset.Interface

	cli rocketcli.Interface

	queue workqueue.RateLimitingInterface
}

func (r *ApplicationReconciler) Run(ctx context.Context) {
	messages := communication.GetChan()
	err := communication.SendMsg(&cache.Message{
		Kind:        "application",
		MessageType: cache.FirstConnect,
	})
	if err != nil {
		log.Error(err)
	}
	go r.run(ctx, 1)
	r.firstSend()
	for msg := range messages {
		if m, ok := msg.(cache.Message); ok {
			if m.Kind == "application" {
				r.queue.Add(&m)
			}
		}
	}
}

func (r *ApplicationReconciler) firstSend() {
	log.Info("first send")
	if err := communication.SendMsg(&cache.Message{
		Kind:        "application",
		MessageType: cache.FirstConnect,
	}); err != nil {
		log.Errorw("first connect send application message error", "error", err)
	}
}

func (r *ApplicationReconciler) run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer r.queue.ShutDown()
	log.Info("starting application workers", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, r.runWorker, time.Second)
	}
	log.Info("started applicatin workers")
	<-ctx.Done()
	log.Info("shutting down application workers")
}

func (r *ApplicationReconciler) runWorker(ctx context.Context) {
	for r.processNextWorkItem(ctx) {
	}
}

func (r *ApplicationReconciler) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := r.queue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer r.queue.Done(obj)
		msg, ok := obj.(*cache.Message)
		if !ok {
			r.queue.Forget(obj)
			log.Errorf("expected *cache.Message in workqueue but got %#v", obj)
			return nil
		}
		if err := r.syncHandler(ctx, msg); err != nil {
			r.queue.AddRateLimited(msg)
			return fmt.Errorf("error syncing %v: %v", msg, err)
		}
		r.queue.Forget(obj)
		log.Infow("successfully synced", "application", msg.Name, "id", msg.ID)
		return nil
	}(obj)
	if err != nil {
		log.Error(err)
	}

	return true
}

func (r *ApplicationReconciler) syncHandler(ctx context.Context, msg *cache.Message) error {
	var app application.Application
	if err := r.cli.Get(fmt.Sprintf("%d", msg.ID), &web.Request{Deleted: true}, &app); err != nil {
		return err
	}
	// application is in current cluster or not
	if app.Cluster != config.Read().Agent.Name {
		log.Debugw("application is not in current cluster", "application", msg.Name, "id", msg.ID)
		return nil
	}
	gvr, workload := GenerateWorkload(&app)
	if msg.ActionType == cache.DeleteAction {
		log.Infow("delete app", "application", msg.Name, "id", msg.ID)
		if !app.DeletedAt.Time.IsZero() {
			// TODO: handle trait delete
			return r.deleteWorkload(ctx, app.Namespace, app.Name, gvr)
		}
		log.Infow("delete application but got deleted_at is null", "application", app.Name, "id", app.ID)
		return nil
	}
	log.Infow("handle application", "application", msg.Name, "id", msg.ID)
	old, err := r.dynamicclientset.Resource(gvr).Namespace(app.Namespace).Get(ctx, app.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infow("resource is not found, create it", "application", app.Name, "namespace", app.Namespace, "id", app.ID)
			if gvr.Resource == "deployments" || gvr.Resource == "clonesets" {
				if err := unstructured.SetNestedField(workload.Object, 1, "spec", "replicas"); err != nil {
					return fmt.Errorf("set application replicas error: %v", err)
				}
			}
			_, err = r.dynamicclientset.Resource(gvr).Namespace(workload.GetNamespace()).Create(ctx, workload, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("create application error: %v", err)
			}
		}
		return fmt.Errorf("get application error: %v", err)
	}
	if gvktools.NeedToUpdate(old, workload) {
		if err := r.setWorkload(old, workload); err != nil {
			return fmt.Errorf("set workload error: %v", err)
		}
		if gvr.Resource == "deployments" || gvr.Resource == "clonesets" {
			if err := unstructured.SetNestedField(workload.Object, 1, "spec", "replicas"); err != nil {
				return fmt.Errorf("set application replicas error: %v", err)
			}
		}
		_, err = r.dynamicclientset.Resource(gvr).Namespace(workload.GetNamespace()).Update(ctx, old, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update application error: %v", err)
		}
	}
	return nil
}

func (r *ApplicationReconciler) setWorkload(old, resource *unstructured.Unstructured) error {
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
	// 3. set anno
	annos, _, _ := unstructured.NestedFieldCopy(resource.Object, "metadata", "annotations")
	if err := unstructured.SetNestedField(old.Object, annos, "metadata", "annotations"); err != nil {
		return fmt.Errorf("set annotations error: %v", err)
	}
	return nil
}

func (r *ApplicationReconciler) deleteWorkload(ctx context.Context, namespace, name string, res schema.GroupVersionResource) error {
	if err := r.dynamicclientset.Resource(res).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) generateObject(app *application.Application, workload *unstructured.Unstructured) (runtime.Object, error) {
	switch app.Template.Type {
	case application.DeploymentType:
		var deployment appsv1.Deployment
		if err := gvktools.ConvertToObject(workload, &deployment); err != nil {
			return &deployment, err
		}
	case application.CloneSetType:
		var cloneset kruiseappsv1alpha1.CloneSet
		if err := gvktools.ConvertToObject(workload, &cloneset); err != nil {
			return &cloneset, err
		}
	case application.CronJobType:
		var cronjob batchv1.CronJob
		if err := gvktools.ConvertToObject(workload, &cronjob); err != nil {
			return &cronjob, err
		}
	}
	return nil, nil
}

// func (c *Controller) getOldTrait(anno map[string]string) mapset.Set[string] {
// 	if anno == nil {
// 		return mapset.NewSet[string]()
// 	}
// 	if v, ok := anno[constant.TraitEdgeAnnotation]; !ok {
// 		return mapset.NewSet[string]()
// 	} else {
// 		set := mapset.NewSet[string]()
// 		for _, s := range strings.Split(v, ",") {
// 			set.Add(s)
// 		}
// 		return set
// 	}
// }

// func (c *Controller) getNewTrait(traits []rocketv1alpha1.Trait) mapset.Set[string] {
// 	set := mapset.NewSet[string]()
// 	for _, t := range traits {
// 		if _, ok := trait.Traits[t.Kind]; ok {
// 			set.Add(t.Kind)
// 		}
// 	}
// 	return set
// }

// func generateRemoveTrait(t []string) []rocketv1alpha1.Trait {
// 	traits := []rocketv1alpha1.Trait{}
// 	for _, v := range t {
// 		switch v {
// 		case trait.PodUnavailableBudgetKind:
// 			traits = append(traits, rocketv1alpha1.Trait{Kind: trait.PodUnavailableBudgetKind})
// 		}
// 	}
// 	return traits
// }

// func (c *Controller) handleTrait(app *rocketv1alpha1.Application) error {
// 	old := c.getOldTrait(app.Annotations)
// 	new := c.getNewTrait(app.Spec.Traits)
// 	removeStr := []string{}
// 	old.Difference(new).Each(func(s string) bool {
// 		removeStr = append(removeStr, s)
// 		return false
// 	})
// 	remove := generateRemoveTrait(removeStr)
// 	if err := c.deleteTrait(remove, app); err != nil {
// 		return err
// 	}
// 	addStr := []string{}
// 	new.Difference(old).Each(func(s string) bool {
// 		addStr = append(addStr, s)
// 		return false
// 	})
// 	return c.createOrUpdateTrait(app.Spec.Traits, app)
// }

// func (c *Controller) deleteTrait(ttmp []rocketv1alpha1.Trait, app *rocketv1alpha1.Application) error {
// 	for _, t := range ttmp {
// 		if opt, ok := trait.Traits[t.Kind]; ok {
// 			client := trait.NewClient(c.kruiseclientset, c.rocketclientset, c.kubeclientset)
// 			if err := opt.Handler(&t, app, trait.Deleted, client); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (c *Controller) createOrUpdateTrait(ttmp []rocketv1alpha1.Trait, app *rocketv1alpha1.Application) error {
// 	for _, t := range ttmp {
// 		if opt, ok := trait.Traits[t.Kind]; ok {
// 			client := trait.NewClient(c.kruiseclientset, c.rocketclientset, c.kubeclientset)
// 			if err := opt.Handler(&t, app, trait.CreatedOrUpdate, client); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }
