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

package application

import (
	"context"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewRecociler(mgr manager.Manager) *ApplicationReconciler {
	return &ApplicationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	app := &rocketv1alpha1.Application{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, app)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(5).Info("application is deleted", "application", tools.KObj(app))
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := app.DeepCopy()
	if obj.DeletionTimestamp.IsZero() {
		if !tools.ContainsString(obj.Finalizers, constant.ApplicationFinalizer) {
			obj.Finalizers = append(obj.Finalizers, constant.ApplicationFinalizer)
			return ctrl.Result{}, r.Update(ctx, obj)
		}
	} else {
		if tools.ContainsString(obj.Finalizers, constant.ApplicationFinalizer) {
			remove := true
			for _, v := range obj.Status.Conditions {
				if v.Type == rocketv1alpha1.ApplicationConditionSuccessedScale {
					switch v.Reason {
					case constant.CloneSetDeleted:
						if v.Status != metav1.ConditionTrue {
							remove = false
							break
						}
					case constant.DeploymentDeleted:
						if v.Status != metav1.ConditionTrue {
							remove = false
							break
						}
					case constant.CronJobDeleted:
						if v.Status != metav1.ConditionTrue {
							remove = false
							break
						}
					}
				}
				if v.Type == rocketv1alpha1.ApplicationConditionSuccessedUpdate {
					if v.Reason != constant.ReasonTraitDeleted {
						if v.Status != metav1.ConditionTrue {
							remove = false
							break
						}

					}
				}
			}
			if remove {
				obj.Finalizers = tools.RemoveString(obj.Finalizers, constant.ApplicationFinalizer)
				if err = r.Update(ctx, obj); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Application{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				old := e.ObjectOld.(*rocketv1alpha1.Application).DeepCopy()
				new := e.ObjectNew.(*rocketv1alpha1.Application).DeepCopy()
				for i := range old.Status.Conditions {
					old.Status.Conditions[i].LastTransitionTime = metav1.Time{}
				}
				for i := range new.Status.Conditions {
					new.Status.Conditions[i].LastTransitionTime = metav1.Time{}
				}
				return !cmp.Equal(old.GetGeneration(), new.GetGeneration()) ||
					!cmp.Equal(old.Status.Conditions, new.Status.Conditions) ||
					!cmp.Equal(old.Labels, new.Labels)
			},
		})).
		Complete(r)
}
