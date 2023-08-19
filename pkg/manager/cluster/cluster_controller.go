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

package cluster

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/controllerrevision"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// 下次 reconciler 间隔时间
const waittime = 5 * time.Minute

// 与上次心跳间隔时间
var timeout time.Duration

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewRecociler(mgr manager.Manager) *ClusterReconciler {
	timeout = time.Duration(config.Pread().TimeOut) * time.Second
	return &ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

//+kubebuilder:rbac:groups=rocket.hextech.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=clusters/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &rocketv1alpha1.Cluster{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: waittime}, err
	}
	obj := cluster.DeepCopy()
	if obj.DeletionTimestamp.IsZero() {
		if !tools.ContainsString(obj.Finalizers, constant.ClusterFinalizer) {
			obj.Finalizers = append(obj.Finalizers, constant.ClusterFinalizer)
			if err = r.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		if tools.ContainsString(obj.Finalizers, constant.ClusterFinalizer) {
			cr := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      obj.Name, // Name is cluster name
					Namespace: constant.RocketNamespace,
				},
			}
			if err := r.Delete(ctx, cr); err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
			obj.Finalizers = tools.RemoveString(obj.Finalizers, constant.ClusterFinalizer)
			if err = r.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	err = r.doReconcile(obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	// predicate 阶段已经确保创建了 controllerrevision 资源
	cr := &appsv1.ControllerRevision{}
	err = r.Get(ctx, types.NamespacedName{Namespace: constant.RocketNamespace, Name: cluster.Name}, cr)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Status().Update(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: waittime}, nil
}

func (r *ClusterReconciler) doReconcile(cluster *rocketv1alpha1.Cluster) error {
	if !cluster.Status.LastKeepAliveTime.IsZero() {
		expire := cluster.Status.LastKeepAliveTime.Add(timeout)
		if time.Now().After(expire) {
			// overtime
			if cluster.Status.State == rocketv1alpha1.Approve {
				cluster.Status.State = rocketv1alpha1.Offline
			}
		} else {
			// recover from offline
			if cluster.Status.State == rocketv1alpha1.Reject || cluster.Status.State == rocketv1alpha1.Pending {
				return nil
			}
			if cluster.Status.State == rocketv1alpha1.Offline {
				cluster.Status.State = rocketv1alpha1.Approve
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := log.FromContext(context.Background())
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Cluster{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				obj := e.Object.(*rocketv1alpha1.Cluster)
				if err := r.handleControllerRevision(obj); err != nil {
					log.Error(err, "create cluster controller revision failed")
				}
				return true
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// 将旧的 cluster 信息，存储到 controllerrevision
				old := ue.ObjectOld.(*rocketv1alpha1.Cluster)
				if err := r.handleControllerRevision(old); err != nil {
					log.Error(err, "update cluster controller revision failed")
				}
				return true
			},
		})).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}

// controllerrevision 资源不能进行更新操作，所有对于 data 字段的更新都会被拒绝
// 当前使用删除重建的方式来处理
func (r *ClusterReconciler) handleControllerRevision(cluster *rocketv1alpha1.Cluster) error {
	cr := &appsv1.ControllerRevision{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: constant.RocketNamespace, Name: cluster.Name}, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			cr = controllerrevision.GenerateCR(cluster)
			return r.Create(context.TODO(), cr)
		}
		return err
	}
	ai := controllerrevision.GenerateAI(cluster)
	b, _ := json.Marshal(ai)
	if !cmp.Equal(cr.Data.Raw, b) {
		err = r.Delete(context.TODO(), cr)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		cr = controllerrevision.GenerateCR(cluster)
		err = r.Create(context.TODO(), cr)
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
