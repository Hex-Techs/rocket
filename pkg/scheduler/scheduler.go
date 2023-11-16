package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	rocketcache "github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/scheduler/cache"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"github.com/hex-techs/rocket/pkg/utils/web"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

// 默认延迟时间
const defaultDelay = 15 * time.Second

// NewReconciler 返回一个新的 reconcile
func NewReconciler() *SchedulerReconciler {
	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)
	return &SchedulerReconciler{
		Interface:      client.New(config.Read().Manager.InternalAddress, config.Read().Manager.AgentBootstrapToken),
		schedCache:     map[string]*clusterCache{},
		snapshot:       map[string]*cache.Snapshot{},
		stopEverything: map[string]chan struct{}{},
		lock:           &sync.RWMutex{},
		queue:          workqueue.NewRateLimitingQueue(ratelimiter),
	}
}

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Interface

	schedCache map[string]*clusterCache

	snapshot map[string]*cache.Snapshot

	lock *sync.RWMutex

	stopEverything map[string]chan struct{}

	queue workqueue.RateLimitingInterface
}

func (r *SchedulerReconciler) Run(ctx context.Context) {
	messages := communication.GetChan()
	err := communication.SendMsg(&rocketcache.Message{
		Kind:        "cluster",
		MessageType: rocketcache.FirstConnect,
	})
	if err != nil {
		log.Error(err)
	}
	go r.run(ctx, 1)
	r.firstSend()
	for msg := range messages {
		if m, ok := msg.(rocketcache.Message); ok {
			if m.Kind == "cluster" {
				r.queue.Add(&m)
			}
		}
	}
}

func (r *SchedulerReconciler) firstSend() {
	log.Info("first send")
	if err := communication.SendMsg(&rocketcache.Message{
		Kind:        "cluster",
		MessageType: rocketcache.FirstConnect,
	}); err != nil {
		log.Error(err)
	}
	// On the first send, we need to wait for the cluster to be ready,
	// so we sleep for 5 seconds before sending the application message.
	time.Sleep(5 * time.Second)
	if err := communication.SendMsg(&rocketcache.Message{
		Kind:        "application",
		MessageType: rocketcache.FirstConnect,
	}); err != nil {
		log.Error(err)
	}
}

func (r *SchedulerReconciler) run(ctx context.Context, workers int) {
	defer r.queue.ShutDown()
	log.Info("starting scheduler workers", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, r.runWorker, time.Second)
	}
	log.Info("started scheduler workers")
	<-ctx.Done()
	log.Info("shutting down scheduler workers")
}

func (r *SchedulerReconciler) runWorker(ctx context.Context) {
	for r.processNextWorkItem(ctx) {
	}
}

func (r *SchedulerReconciler) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := r.queue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer r.queue.Done(obj)
		var key *rocketcache.Message
		var ok bool
		if key, ok = obj.(*rocketcache.Message); !ok {
			r.queue.Forget(obj)
			log.Errorf("expected *cache.Message in workqueue but got %#v", obj)
			return nil
		}
		if err := r.syncHandler(ctx, key); err != nil {
			r.queue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%v': %v, requeuing", key, err)
		}
		r.queue.Forget(obj)
		log.Infof("successfully synced '%v'", key)
		return nil
	}(obj)
	if err != nil {
		log.Error(err)
	}
	return true
}

func (r *SchedulerReconciler) syncHandler(ctx context.Context, msg *rocketcache.Message) error {
	if msg.Kind == "cluster" {
		return r.doCluster(ctx, msg)
	}
	if msg.Kind == "application" {
		return r.doApplication(ctx, msg)
	}
	return nil
}

func (r *SchedulerReconciler) doCluster(ctx context.Context, msg *rocketcache.Message) error {
	cls := &cluster.Cluster{}
	if msg.ActionType == rocketcache.DeleteAction {
		r.removeCluster(msg.Name)
		return nil
	}
	if err := r.Get(msg.Name, &web.Request{Namespace: msg.Namespace}, cls); err != nil {
		if errors.IsNotFound(err) {
			log.Warnw("cluster is not found", "clusterName", msg.Name, "namespace", msg.Namespace)
			r.removeCluster(msg.Name)
			return nil
		}
		return err
	}
	if msg.ActionType == rocketcache.CreateAction {
		r.addCluster(cls)
	}
	if msg.ActionType == rocketcache.UpdateAction {
		r.updateCluster(cls)
	}
	return nil
}

func (r *SchedulerReconciler) doApplication(ctx context.Context, msg *rocketcache.Message) error {
	app := &application.Application{}
	if msg.ActionType == rocketcache.DeleteAction {
		return nil
	}
	if err := r.Get(msg.Name, &web.Request{Namespace: msg.Namespace}, app); err != nil {
		if errors.IsNotFound(err) {
			log.Warnw("application is not found", "applicationName", msg.Name, "namespace", msg.Namespace)
			return nil
		}
		return err
	}
	if app.Phase != application.Scheduling && app.Phase != application.Descheduling {
		log.Debugw("application is not in Scheduling or Descheduling phase", "application", tools.KObj(app), "phase", app.Phase)
		return nil
	}
	if r.doSchedule(ctx, app) {
		app.Phase = application.Scheduled
		if err := r.Update(fmt.Sprintf("%d", app.ID), app); err != nil {
			return err
		}
	} else {
		if app.Phase != application.Scheduled {
			// 延迟15s重新调度
			r.queue.AddAfter(msg, defaultDelay)
		}
	}
	return nil
}

func (r *SchedulerReconciler) doSchedule(ctx context.Context, application *application.Application) bool {
	schedulingCycleCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := r.scheduler(schedulingCycleCtx, application)
	if len(result) == 0 {
		log.Info("unable to scheduler application because no cluster can be scheduled", "application", tools.KObj(application))
		return false
	}

	// 已经调度的 application 保留其上一次调度的信息，如果上一次结果和本次相同，则不做更新，保留上次的结果
	if len(application.Cluster) != 0 {
		application.LastSchedulerClusters = application.Cluster
	}
	// result already sorted by score, use the first one in application status
	application.Cluster = result[0].Name
	log.Info("application was schedule sucessed", "application", tools.KObj(application), "cluster", application.Cluster)
	return true
}

func (r *SchedulerReconciler) addCluster(cls *cluster.Cluster) {
	_, ok := r.schedCache[cls.Name]
	if ok {
		log.Infow("cluster is already exist", "clusterName", cls.Name)
		return
	}
	kcli, err := generateClient(cls)
	if err != nil {
		log.Errorw("unable to add cluster to scheduler", "clusterName", cls.Name, "error", err)
		return
	}
	log.Infow("add cluster to scheduler cache", "clusterName", cls.Name)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kcli, time.Second*120)
	c := NewClusterCache(cls, kcli, kubeInformerFactory.Core().V1().Nodes(), kubeInformerFactory.Core().V1().Pods())
	r.snapshot[cls.Name] = cache.NewEmptySnapshot()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.schedCache[cls.Name] = c
	r.stopEverything[cls.Name] = make(chan struct{})
	kubeInformerFactory.Start(r.stopEverything[cls.Name])
}

func (r *SchedulerReconciler) updateCluster(cls *cluster.Cluster) {
	_, ok := r.schedCache[cls.Name]
	if !ok {
		log.Infow("cluster is not exist", "clusterName", cls.Name)
		return
	}
	kcli, err := generateClient(cls)
	if err != nil {
		log.Errorw("ubable to update cluster to scheduler", "clusterName", cls.Name, "error", err)
		return
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kcli, time.Second*120)
	c := NewClusterCache(cls, kcli, kubeInformerFactory.Core().V1().Nodes(), kubeInformerFactory.Core().V1().Pods())
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.schedCache, cls.Name)
	delete(r.snapshot, cls.Name)
	close(r.stopEverything[cls.Name])
	delete(r.stopEverything, cls.Name)

	r.schedCache[cls.Name] = c
	r.snapshot[cls.Name] = cache.NewEmptySnapshot()
	r.stopEverything[cls.Name] = make(chan struct{})
	kubeInformerFactory.Start(r.stopEverything[cls.Name])
}

func (r *SchedulerReconciler) removeCluster(name string) {
	log.Infow("cluster is deleted", "clusterName", name)
	r.lock.Lock()
	defer r.lock.Unlock()
	// TODO: will close client
	delete(r.schedCache, name)
	delete(r.snapshot, name)
	close(r.stopEverything[name])
	delete(r.stopEverything, name)
}

func generateClient(cls *cluster.Cluster) (kubernetes.Interface, error) {
	config, err := clustertools.GenerateRestConfigFromCluster(cls)
	if err != nil {
		return nil, err
	}
	kcli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return kcli, nil
}
