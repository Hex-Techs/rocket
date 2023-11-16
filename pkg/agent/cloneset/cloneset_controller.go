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
	"encoding/json"

	"github.com/fize/go-ext/log"
	rocketcli "github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const CloneSetKind = "cloneset"

func NewCloneSetReconciler(mgr manager.Manager) *CloneSetReconciler {
	return &CloneSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		cli:    rocketcli.New(config.Read().Manager.InternalAddress, config.Read().Manager.AgentBootstrapToken),
	}
}

// CloneSetReconciler reconciles a CloneSet object
type CloneSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	cli    rocketcli.Interface
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloneSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cs := &kruiseappsv1alpha1.CloneSet{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, cs); err != nil {
		if errors.IsNotFound(err) {
			log.Debugw("cloneset has been deleted", "cloneset", req)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	appID := cs.Labels[constant.AppIDLabel]
	if appID == "" {
		log.Debugw("cloneset has no appID label", "cronjob", req)
		return ctrl.Result{}, nil
	}
	var app application.Application
	if err := r.cli.Get(appID, nil, &app); err != nil {
		log.Errorw("get application error", "cloneset", req, "error", err)
	}
	raw, _ := json.Marshal(cs.Status)
	app.ApplicationDetails = string(raw)
	if err := r.cli.Update(appID, &app); err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloneSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kruiseappsv1alpha1.CloneSet{}).
		Complete(r)
}
