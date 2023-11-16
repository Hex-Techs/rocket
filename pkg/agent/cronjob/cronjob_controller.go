package cronjob

import (
	"context"
	"encoding/json"

	"github.com/fize/go-ext/log"
	rocketcli "github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const CronJobKind = "cronjob"

// NewReconcile 返回一个新的 reconcile
func NewCronJobReconcile(mgr manager.Manager) *CronJobReconciler {
	return &CronJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		cli:    rocketcli.New(config.Read().Manager.InternalAddress, config.Read().Manager.AgentBootstrapToken),
	}
}

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	cli    rocketcli.Interface
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cj := &batchv1.CronJob{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, cj); err != nil {
		if errors.IsNotFound(err) {
			log.Debugw("cronjob has been deleted", "cronjob", req)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	appID := cj.Labels[constant.AppIDLabel]
	if appID == "" {
		log.Debugw("cronjob has no appID label", "cronjob", req)
		return ctrl.Result{}, nil
	}
	var app application.Application
	if err := r.cli.Get(appID, nil, &app); err != nil {
		log.Errorw("get application error", "cronjob", req, "error", err)
	}
	raw, _ := json.Marshal(cj.Status)
	app.ApplicationDetails = string(raw)
	if err := r.cli.Update(appID, &app); err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Complete(r)
}
