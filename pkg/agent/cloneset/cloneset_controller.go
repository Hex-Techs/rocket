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

package cloneset

import (
	"context"
	"fmt"
	"time"

	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/util/clustertools"
	"github.com/hex-techs/rocket/pkg/util/condition"
	"github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const CloneSetKind = "cloneset"

func NewCloneSetReconciler(mgr manager.Manager) *CloneSetReconciler {
	cfg, err := clustertools.GenerateKubeConfigFromToken(config.Pread().MasterURL,
		config.Pread().BootstrapToken, nil, 1)
	if err != nil {
		klog.Fatal(err)
	}
	return &CloneSetReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		rclient: clientset.NewForConfigOrDie(cfg),
	}
}

// CloneSetReconciler reconciles a CloneSet object
type CloneSetReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	rclient clientset.Interface
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloneSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// rocket创建的cloneset，其名称前缀为rocket-，其对应的workload名称为其名称去掉前缀
	if len(req.Name) < 5 {
		klog.V(3).Infof("%s without rocket needs no attention", req)
		return ctrl.Result{}, nil
	}
	wname := req.Name[5:]
	workload, err := r.rclient.RocketV1alpha1().Workloads(req.Namespace).Get(ctx, wname, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(1).Infof("can not found Workload '%s', skip this CloneSet '%s'", wname, req)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := workload.DeepCopy()
	cs := &kruiseappsv1alpha1.CloneSet{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, cs)
	if err != nil {
		if errors.IsNotFound(err) {
			if len(obj.Status.Conditions) == 0 {
				obj.Status.Conditions = make(map[string]metav1.Condition)
			}
			obj.Status.Conditions[config.Pread().Name] = condition.GenerateCondition("CloneSet", "CloneSet",
				fmt.Sprintf("%s has been deleted", req), metav1.ConditionTrue)
			obj.Status.ClonSetStatus = nil
			_, err = r.rclient.RocketV1alpha1().Workloads(req.Namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if w, ok := cs.Labels[constant.WorkloadNameLabel]; ok {
		if wname == w {
			if len(obj.Status.Conditions) == 0 {
				obj.Status.Conditions = make(map[string]metav1.Condition)
			}
			obj.Status.Conditions[config.Pread().Name] = condition.GenerateCondition("CloneSet", "CloneSet",
				fmt.Sprintf("%s create successed", req), metav1.ConditionTrue)
			obj.Status.ClonSetStatus = &cs.Status
			_, err = r.rclient.RocketV1alpha1().Workloads(req.Namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloneSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kruiseappsv1alpha1.CloneSet{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}
