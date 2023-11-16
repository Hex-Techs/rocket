package hubcluster

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/agent/communication"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/cache/localcache"
	"github.com/hex-techs/rocket/pkg/client"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/clustertools"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"golang.org/x/time/rate"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

func NewReconciler() *ClusterReconciler {
	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)
	return &ClusterReconciler{
		cli:   client.New(config.Read().Manager.InternalAddress, config.Read().Manager.AgentBootstrapToken),
		queue: workqueue.NewRateLimitingQueue(ratelimiter),
	}
}

type ClusterReconciler struct {
	cli   client.Interface
	queue workqueue.RateLimitingInterface
}

func (r *ClusterReconciler) Run(ctx context.Context) {
	messages := communication.GetChan()
	err := communication.SendMsg(&cache.Message{
		Kind:        "cluster",
		MessageType: cache.FirstConnect,
	})
	if err != nil {
		log.Error(err)
	}
	go r.run(ctx, 1)
	r.firstSend()
	for msg := range messages {
		if m, ok := msg.(cache.Message); ok {
			if m.Kind == "cluster" {
				r.queue.Add(&m)
			}
		}
	}
}

func (r *ClusterReconciler) firstSend() {
	log.Info("first send")
	if err := communication.SendMsg(&cache.Message{
		Kind:        "cluster",
		MessageType: cache.FirstConnect,
	}); err != nil {
		log.Errorw("first connect send cluster message error", "error", err)
	}
}

func (r *ClusterReconciler) run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer r.queue.ShutDown()
	log.Info("starting cluster workers", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, r.runWorker, time.Second)
	}
	log.Info("started cluster workers")
	<-ctx.Done()
	log.Info("shutting down cluster workers")
}

func (r *ClusterReconciler) runWorker(ctx context.Context) {
	for r.processNextWorkItem(ctx) {
	}
}

func (r *ClusterReconciler) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := r.queue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer r.queue.Done(obj)
		var key *cache.Message
		var ok bool
		if key, ok = obj.(*cache.Message); !ok {
			r.queue.Forget(obj)
			log.Errorf("expected *cache.Message in workqueue but got %#v", obj)
			return nil
		}
		if err := r.syncHandler(ctx, key); err != nil {
			r.queue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %v, requeuing", key.Name, err)
		}
		r.queue.Forget(obj)
		log.Infow("Successfully synced", "cluster", key.Name, "resourceKind", key.Kind)
		return nil
	}(obj)

	if err != nil {
		log.Error(err)
	}

	return true
}

func (r *ClusterReconciler) syncHandler(ctx context.Context, msg *cache.Message) error {
	var cluster cluster.Cluster
	if msg.ActionType == cache.DeleteAction {
		log.Infow("delete local cache", "cluster", msg.Name, "id", msg.ID)
		localcache.DeleteClient(msg.ID)
		return nil
	}
	if err := r.cli.Get(fmt.Sprintf("%d", msg.ID), nil, &cluster); err != nil {
		return err
	}
	if cluster.Agent {
		log.Debugw("cluster is agent, skip", "cluster", cluster.Name)
		return nil
	}
	log.Infow("handle local cache", "cluster", cluster.Name, "id", cluster.ID)
	if msg.ActionType == cache.CreateAction {
		_, ok := localcache.GetClient(msg.ID)
		if ok {
			localcache.DeleteClient(msg.ID)
		}
	}
	if msg.ActionType == cache.UpdateAction {
		if tmp, ok := localcache.GetClient(msg.ID); ok {
			cfg, err := clustertools.GenerateRestConfigFromCluster(&cluster)
			if err != nil {
				return err
			}
			if reflect.DeepEqual(tmp.GetKubeConfig(), cfg) {
				log.Debugw("kubeconfig is equal, skip", "cluster", cluster.Name, "id", msg.ID)
				return nil
			} else {
				log.Debugw("kubeconfig is not equal, delete old", "cluster", cluster.Name, "id", msg.ID)
				localcache.DeleteClient(msg.ID)
				return nil
			}
		}
	}
	return r.do(&cluster)
}

// do create local cache
func (r *ClusterReconciler) do(cls *cluster.Cluster) error {
	c, err := localcache.NewClusterCache(cls)
	if err != nil {
		return err
	}
	localcache.AddClient(c)
	return nil
}
