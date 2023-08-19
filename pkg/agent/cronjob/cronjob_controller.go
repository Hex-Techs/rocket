package cronjob

import (
	"context"
	"encoding/json"
	"os"
	"time"

	clientset "github.com/hex-techs/rocket/client/clientset/versioned"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"golang.org/x/time/rate"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const CronJobKind = "cronjob"

const (
	// CronJobCreated means cronjob has been created
	CronJobCreated = "CronJobCreated"
	// CronJobDeleted means cronjob has been deleted
	CronJobDeleted = "CronJobDeleted"
)

// NewReconcile 返回一个新的 reconcile
func NewCronJobReconcile(mgr manager.Manager) *CronJobReconciler {
	cfg, err := clustertools.GenerateKubeConfigFromToken(config.Pread().MasterURL,
		config.Pread().BootstrapToken, nil, 1)
	if err != nil {
		log.Log.Error(err, "generate kubeconfig failed")
		os.Exit(1)
	}
	return &CronJobReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		rclient: clientset.NewForConfigOrDie(cfg),
	}
}

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	rclient clientset.Interface
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	obj, err := r.rclient.RocketV1alpha1().Applications(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(5).Info("can not found Workload skip this CronJob", "cronjob", req)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	app := obj.DeepCopy()
	cj := &batchv1.CronJob{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, cj)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(5).Info("deployment has been deleted", "cronjob", req)
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
	obj.Status.ApplicationDetails.Raw, _ = json.Marshal(cj.Status)
	_, err = r.rclient.RocketV1alpha1().Applications(req.Namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}
