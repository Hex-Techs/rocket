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

package distribution

import (
	"context"
	"time"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// The resource is ready or not
	ResourceReadyReason = "ResourceReady"

	// The resource is deleted
	ResourceDeleteReason = "ResourceDeleted"
)

// NewReconciler 返回一个新的 reconcile
func NewReconciler(mgr manager.Manager) *DistributionReconciler {
	return &DistributionReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

// distributionReconciler reconciles a distribution object
type DistributionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rocket.hextech.io,resources=distributions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=distributions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=distributions/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DistributionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	distribution := &rocketv1alpha1.Distribution{}
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, distribution)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := distribution.DeepCopy()
	return r.doReconcile(ctx, obj)
}

// 执行 reconcile
func (r *DistributionReconciler) doReconcile(ctx context.Context, obj *rocketv1alpha1.Distribution) (ctrl.Result, error) {
	if obj.DeletionTimestamp.IsZero() {
		if obj.Finalizers == nil {
			obj.Finalizers = []string{}
		}
		if !tools.ContainsString(obj.Finalizers, constant.DistributionFinalizer) {
			obj.Finalizers = append(obj.Finalizers, constant.DistributionFinalizer)
			if err := r.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if tools.ContainsString(obj.Finalizers, constant.DistributionFinalizer) {
			if len(obj.Status.Conditions) > 0 {
				for _, condition := range obj.Status.Conditions {
					if condition.Reason != ResourceDeleteReason {
						return ctrl.Result{}, nil
					}
					if condition.Status != metav1.ConditionTrue {
						return ctrl.Result{}, nil
					}
				}
			}
			obj.Finalizers = tools.RemoveString(obj.Finalizers, constant.DistributionFinalizer)
			if err := r.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DistributionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Distribution{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}
