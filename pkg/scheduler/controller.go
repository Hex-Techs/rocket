package scheduler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/clustertools"
	"github.com/hex-techs/rocket/pkg/util/constant"
	"github.com/hex-techs/rocket/pkg/util/controllerrevision"
	"github.com/hex-techs/rocket/pkg/util/signals"
	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// 默认延迟时间
const defaultDelay = 15 * time.Second

// NewReconcile 返回一个新的 reconcile
func NewReconcile(mgr manager.Manager) *SchedulerReconciler {
	return &SchedulerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workload := &rocketv1alpha1.Workload{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, workload)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := workload.DeepCopy()
	if r.syncHandler(obj) {
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

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Workload{}).
		Watches(source.NewKindWithCache(&rocketv1alpha1.Cluster{}, mgr.GetCache()), handler.Funcs{
			UpdateFunc: func(ue event.UpdateEvent, q workqueue.RateLimitingInterface) {
				obj := ue.ObjectNew.(*rocketv1alpha1.Cluster)
				if obj.Status.State == rocketv1alpha1.Approve {
					cr := &appsv1.ControllerRevision{}
					err := r.Get(context.TODO(), types.NamespacedName{Namespace: constant.RocketNamespace, Name: obj.Name}, cr)
					if err != nil {
						klog.Errorf("get controller revision error: %v", err)
						return
					}
					r.handleAgentCluster(obj, cr)
				} else {
					r.closeClient(obj.Name)
				}
			},
			DeleteFunc: func(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
				r.closeClient(e.Object.GetName())
			},
		}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}
func (r *SchedulerReconciler) syncHandler(workload *rocketv1alpha1.Workload) bool {
	if workload.Status.Phase != "Scheduling" {
		// 如果想要重新调度，则修改这个状态就可以了
		return false
	}
	cr := Scheduler(workload)
	result := map[string][]int64{}
	for idx, v := range cr {
		if s, ok := result[v.Region]; !ok {
			result[v.Region] = []int64{}
			result[v.Region] = append(result[v.Region], int64(idx))
			result[v.Region] = append(result[v.Region], v.Score)
		} else {
			if s[1] < v.Score {
				result[v.Region][0] = int64(idx)
				result[v.Region][1] = v.Score
			}
		}
	}
	if len(result) == 0 {
		klog.V(3).Infof("workload %s/%s no cluster can be scheduled", workload.Namespace, workload.Name)
		return false
	}
	// 已经调度的 workload 保留其上一次调度的信息，如果上一次结果和本次相同，则不做更新，保留上次的结果
	if len(workload.Status.Clusters) != 0 {
		if len(workload.Annotations) == 0 {
			workload.Annotations = map[string]string{}
		}
		cs := strings.Join(workload.Status.Clusters, ",")
		if v, ok := workload.Annotations[constant.LastSchedulerClusterAnnotation]; ok {
			if v != cs {
				workload.Annotations[constant.LastSchedulerClusterAnnotation] = cs
			}
		} else {
			workload.Annotations[constant.LastSchedulerClusterAnnotation] = cs
		}
	}
	workload.Status.Clusters = []string{}
	for _, v := range result {
		workload.Status.Clusters = append(workload.Status.Clusters, cr[int(v[0])].Name)
	}
	klog.V(3).Infof("workload '%s/%s' was schedule sucessed in cluster %v", workload.Namespace, workload.Name, workload.Status.Clusters)
	return true
}
func (r *SchedulerReconciler) handleAgentCluster(cluster *rocketv1alpha1.Cluster, cr *appsv1.ControllerRevision) {
	config, err := clustertools.GenerateRestConfigFromCluster(cluster)
	if err != nil {
		klog.Errorf("generate '%s' rest config with error: %v", cluster.Name, err)
		// 如果获取配置文件错误，那么就认为之前的client也已经失效，删除相应的client
		ClientMap.Delete(cluster.Name)
		return
	}
	var cc *ClusterClient
	kcli, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("generate '%s' kubernetes client with error: %v", cluster.Name, err)
		ClientMap.Delete(cluster.Name)
		return
	}
	dcli, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Errorf("generate '%s' dynamic client with error: %v", cluster.Name, err)
		ClientMap.Delete(cluster.Name)
		return
	}
	cli, ok := ClientMap.Load(cluster.Name)
	if !ok {
		cc = &ClusterClient{
			Region:        cluster.Spec.Region,
			Area:          cluster.Spec.Area,
			Env:           cluster.Spec.Environment,
			Taint:         cluster.Spec.Taints,
			Client:        kcli,
			DynamicClient: dcli,
			Stop:          make(chan struct{}),
		}
	} else {
		cc = cli.(*ClusterClient)
		ai := controllerrevision.GenerateAI(cluster)
		b, _ := json.Marshal(ai)
		if cmp.Equal(cr.Data.Raw, b) {
			klog.V(4).Infof("cluster '%s' auth not change, skip init client", cluster.Name)
			return
		}
		close(cc.Stop)
		cc.Stop = make(chan struct{})
		cc.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			klog.Errorf("cluster '%s' init client with error: %v", cluster.Name, err)
			return
		}
		cc.DynamicClient, err = dynamic.NewForConfig(config)
		if err != nil {
			klog.Errorf("cluster '%s' init dynamic client with error: %v", cluster.Name, err)
			return
		}
	}
	cc.StopCh = signals.SetupSignalHandler(cc.Stop)
	if !cmp.Equal(cluster.Spec.Taints, cc.Taint) {
		cc.Taint = cluster.Spec.Taints
	}
	ClientMap.Store(cluster.Name, cc)
	go r.handleClusterSync(cluster.Name)
}
func (r *SchedulerReconciler) handleClusterSync(name string) {
	cli, ok := ClientMap.Load(name)
	if !ok {
		klog.Errorf("client not found '%s' when handle cluster sync", name)
	}
	cc := cli.(*ClusterClient)
	klog.V(0).Infof("client found: clustername: %s, client: %v", name, cc)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(cc.Client, time.Second*30)
	controller := NewCacheController(name, cc.Client, kubeInformerFactory.Core().V1().Nodes(), kubeInformerFactory.Core().V1().Pods())
	cc.CacheController = controller
	kubeInformerFactory.Start(cc.StopCh)
	if err := controller.Run(name, 2, cc.StopCh); err != nil {
		klog.Errorf("error running controller: %s", err)
	}
}
func (r *SchedulerReconciler) closeClient(name string) {
	klog.V(3).Infof("%s cluster is being deleted", name)
	cli, ok := ClientMap.Load(name)
	if !ok {
		klog.Warningf("%s client not found", name)
		return
	}
	cc := cli.(*ClusterClient)
	close(cc.Stop)
	ClientMap.Delete(name)
}
