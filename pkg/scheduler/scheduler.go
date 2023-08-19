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

package scheduler

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/scheduler/cache"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// 默认延迟时间
const defaultDelay = 15 * time.Second

// NewReconcile 返回一个新的 reconcile
func NewReconcile(mgr manager.Manager) *SchedulerReconciler {
	return &SchedulerReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		schedCache:     map[string]*clusterCache{},
		snapshot:       map[string]*cache.Snapshot{},
		stopEverything: map[string]chan struct{}{},
		lock:           &sync.RWMutex{},
		recoder:        mgr.GetEventRecorderFor("liuer-scheduler"),
	}
}

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	schedCache map[string]*clusterCache

	snapshot map[string]*cache.Snapshot

	lock *sync.RWMutex

	stopEverything map[string]chan struct{}

	recoder record.EventRecorder
}

//+kubebuilder:rbac:groups=liuer.yangshipin.com,resources=Schedulers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=liuer.yangshipin.com,resources=Schedulers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=liuer.yangshipin.com,resources=Schedulers/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	application := &rocketv1alpha1.Application{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, application)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := application.DeepCopy()
	if obj.Status.Phase != "Scheduling" {
		log.V(0).Info("application is not in scheduling phase", "application", tools.KObj(obj), "phase", obj.Status.Phase)
		return reconcile.Result{}, nil
	}
	if r.doReconcile(ctx, obj) {
		obj.Status.Phase = "Running"
		if err := r.Status().Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if obj.Status.Phase != "Running" {
			// 延迟15s重新调度
			return ctrl.Result{RequeueAfter: defaultDelay}, nil
		}
	}
	return reconcile.Result{}, nil
}

func (r *SchedulerReconciler) doReconcile(ctx context.Context, application *rocketv1alpha1.Application) bool {
	log := log.FromContext(context.Background())
	schedulingCycleCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.recoder.Eventf(application, v1.EventTypeNormal, "Scheduling", "Start scheduling application %s/%s", application.Namespace, application.Name)
	result := r.scheduler(schedulingCycleCtx, application)
	if len(result) == 0 {
		log.Error(nil, "unable to scheduler application", "application", tools.KObj(application))
		r.recoder.Eventf(application, v1.EventTypeWarning, "SchedulerFailed",
			"unable to scheduler application '%s/%s' because no cluster can be scheduled", application.Namespace, application.Name)
		return false
	}

	// 已经调度的 application 保留其上一次调度的信息，如果上一次结果和本次相同，则不做更新，保留上次的结果
	if len(application.Status.Clusters) != 0 {
		if len(application.Annotations) == 0 {

			application.Annotations = map[string]string{}
		}
		cs := strings.Join(application.Status.Clusters, ",")
		if v, ok := application.Annotations[constant.LastSchedulerClusterAnnotation]; ok {
			if v != cs {
				application.Annotations[constant.LastSchedulerClusterAnnotation] = cs
			}
		} else {
			application.Annotations[constant.LastSchedulerClusterAnnotation] = cs
		}
	}
	// result already sorted by score, use the first one in application status
	// NOTE: only one cluster can be scheduled
	application.Status.Clusters = []string{result[0].Name}
	log.V(3).Info("application was schedule sucessed", "application", tools.KObj(application), "cluster", application.Status.Clusters)
	r.recoder.Eventf(application, v1.EventTypeNormal, "Scheduling",
		"Application '%s/%s' was schedule sucessed in cluster '%v'", application.Namespace, application.Name, application.Status.Clusters)
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Application{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				new := e.ObjectNew.(*rocketv1alpha1.Application).DeepCopy()
				return new.Status.Phase == "Scheduling"
			},
		})).
		Watches(source.NewKindWithCache(&rocketv1alpha1.Cluster{}, mgr.GetCache()), handler.Funcs{
			CreateFunc: func(e event.CreateEvent, q workqueue.RateLimitingInterface) {
				obj := e.Object.(*rocketv1alpha1.Cluster).DeepCopy()
				if obj.Status.State == rocketv1alpha1.Approve {
					r.addCluster(obj)
				}
			},
			// cluter在创建时，一般状态为pending，需要等待审批通过后，才能创建client
			UpdateFunc: func(ue event.UpdateEvent, q workqueue.RateLimitingInterface) {
				new := ue.ObjectNew.(*rocketv1alpha1.Cluster).DeepCopy()
				old := ue.ObjectOld.(*rocketv1alpha1.Cluster).DeepCopy()
				if new.Status.State == rocketv1alpha1.Approve {
					if !cmp.Equal(new.GetGeneration(), old.GetGeneration()) {
						r.updateCluster(new)
					}
				} else {
					r.removeCluster(new.Name)
				}
			},
			DeleteFunc: func(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
				r.removeCluster(e.Object.GetName())
			},
		}).
		Watches(source.NewKindWithCache(&rocketv1alpha1.Application{}, mgr.GetCache()), handler.Funcs{}).
		WithOptions(controller.Options{}).Complete(r)
}

func (r *SchedulerReconciler) addCluster(cluster *rocketv1alpha1.Cluster) {
	log := log.FromContext(context.Background())
	_, ok := r.schedCache[cluster.Name]
	if ok {
		log.V(3).Info("cluster is already exist", "clusterName", cluster.Name)
		return
	}
	kcli, err := generateClient(cluster)
	if err != nil {
		log.Error(err, "unable to add cluster to scheduler", "clusterName", cluster.Name)
		return
	}
	log.V(0).Info("add cluster to scheduler cache", "clusterName", cluster.Name)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kcli, time.Second*120)
	c := NewClusterCache(cluster, kcli, kubeInformerFactory.Core().V1().Nodes(), kubeInformerFactory.Core().V1().Pods())
	r.snapshot[cluster.Name] = cache.NewEmptySnapshot()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.schedCache[cluster.Name] = c
	r.stopEverything[cluster.Name] = make(chan struct{})
	kubeInformerFactory.Start(r.stopEverything[cluster.Name])
	// go wait.Until(func() {
	// 	for {
	// 		time.Sleep(time.Second)
	// 	}
	// }, time.Second, stop)
}

func (r *SchedulerReconciler) updateCluster(cluster *rocketv1alpha1.Cluster) {
	log := log.FromContext(context.Background())
	_, ok := r.schedCache[cluster.Name]
	if !ok {
		log.V(3).Info("cluster is not exist", "clusterName", cluster.Name)
		return
	}
	kcli, err := generateClient(cluster)
	if err != nil {
		log.Error(err, "ubable to update cluster to scheduler", "clusterName", cluster.Name)
		return
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kcli, time.Second*120)
	c := NewClusterCache(cluster, kcli, kubeInformerFactory.Core().V1().Nodes(), kubeInformerFactory.Core().V1().Pods())
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.schedCache, cluster.Name)
	delete(r.snapshot, cluster.Name)
	close(r.stopEverything[cluster.Name])
	delete(r.stopEverything, cluster.Name)

	r.schedCache[cluster.Name] = c
	r.snapshot[cluster.Name] = cache.NewEmptySnapshot()
	r.stopEverything[cluster.Name] = make(chan struct{})
	kubeInformerFactory.Start(r.stopEverything[cluster.Name])
}

func (r *SchedulerReconciler) removeCluster(name string) {
	log := log.FromContext(context.Background())
	log.V(3).Info("cluster is deleted", "clusterName", name)
	r.lock.Lock()
	defer r.lock.Unlock()
	// TODO: will close client
	delete(r.schedCache, name)
	delete(r.snapshot, name)
	close(r.stopEverything[name])
	delete(r.stopEverything, name)
}

func generateClient(cluster *rocketv1alpha1.Cluster) (kubernetes.Interface, error) {
	config, err := clustertools.GenerateRestConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	kcli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return kcli, nil
}
