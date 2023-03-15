/*
Copyright 2023.

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

package template

import (
	"context"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TemplateReconciler reconciles a Template object
type TemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewRecociler(mgr ctrl.Manager) *TemplateReconciler {
	return &TemplateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

//+kubebuilder:rbac:groups=rocket.hextech.io,resources=templates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=templates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=templates/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	template := &rocketv1alpha1.Template{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, template)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := template.DeepCopy()
	status := rocketv1alpha1.TemplateStatus{
		UsedAgain:        metav1.ConditionFalse,
		NumberOfWorkload: 0,
		UsedBy:           []rocketv1alpha1.UsedBy{},
	}
	if template.Spec.ApplyScope.AllowOverlap {
		status.UsedAgain = metav1.ConditionTrue
	}

	acs := &rocketv1alpha1.ApplicationList{}
	err = r.List(ctx, acs, &client.ListOptions{Namespace: req.Namespace})
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, ac := range acs.Items {
		if r.includeApp(req.Name, &ac) {
			status.UsedBy = append(status.UsedBy, rocketv1alpha1.UsedBy{Name: ac.Name, UID: string(ac.UID)})
			status.NumberOfWorkload = status.NumberOfWorkload + 1
		}
	}
	if !cmp.Equal(obj.Status, status) {
		obj.Status = status
		if err := r.Status().Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *TemplateReconciler) includeApp(cname string, app *rocketv1alpha1.Application) bool {
	for _, a := range app.Spec.Templates {
		if a.Name == cname {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *TemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Template{}).
		Watches(source.NewKindWithCache(&rocketv1alpha1.Application{}, mgr.GetCache()), handler.Funcs{
			CreateFunc: r.handleApplicationCreated,
			UpdateFunc: r.handleApplicationUpdated,
			DeleteFunc: r.handleApplicationDeleted,
		}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}

// handle the workload was created
func (r *TemplateReconciler) handleApplicationCreated(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.handleApplication(event.GenericEvent{Object: e.Object}, q)
}

// handle the workload was updated
func (r *TemplateReconciler) handleApplicationUpdated(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.handleApplication(event.GenericEvent{Object: e.ObjectNew}, q)
}

// handle the workload was deleted
func (r *TemplateReconciler) handleApplicationDeleted(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.handleApplication(event.GenericEvent{Object: e.Object}, q)
}

func (r *TemplateReconciler) handleApplication(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	l := e.Object.GetAnnotations()
	if s, ok := l[constant.TemplateUsed]; !ok {
		klog.Warningf("not found templates by annotation '%s'", constant.TemplateUsed)
		return
	} else {
		for _, n := range strings.Split(s, ",") {
			q.Add(ctrl.Request{NamespacedName: types.NamespacedName{Namespace: e.Object.GetNamespace(), Name: n}})
		}
	}
}
