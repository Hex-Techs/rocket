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

package deployment

import (
	"context"
	"encoding/json"
	"os"

	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const DeploymentKind = "deployment"

const (
	// DeploymentCreated means deployment has been created
	DeploymentCreated = "DeploymentCreated"
	// DeploymentDeleted means deployment has been deleted
	DeploymentDeleted = "DeploymentDeleted"
)

func NewDeploymentReconciler(mgr manager.Manager) *DeploymentReconciler {
	cfg, err := clustertools.GenerateKubeConfigFromToken(config.Pread().MasterURL,
		config.Pread().BootstrapToken, nil, 1)
	if err != nil {
		log.Log.Error(err, "generate kubeconfig failed")
		os.Exit(1)
	}
	return &DeploymentReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		rclient: clientset.NewForConfigOrDie(cfg),
	}
}

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	rclient clientset.Interface
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	obj, err := r.rclient.RocketV1alpha1().Applications(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(5).Info("can not found Workload skip this Deployment", "deployment", req)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	app := obj.DeepCopy()
	deploy := &appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(5).Info("deployment has been deleted", "deployment", req)
			// deployment has been deleted but application still exists
			app.Status.ApplicationDetails = &runtime.RawExtension{}
			_, err = r.rclient.RocketV1alpha1().Applications(req.Namespace).UpdateStatus(ctx, app, metav1.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj.Status.ApplicationDetails = &runtime.RawExtension{}
	obj.Status.ApplicationDetails.Raw, _ = json.Marshal(deploy.Status)
	_, err = r.rclient.RocketV1alpha1().Applications(req.Namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
