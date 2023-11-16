package edgecluster

import (
	"context"
	"fmt"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/google/go-cmp/cmp"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const ClusterKind = "cluster"

func NewReconciler(mgr manager.Manager) *ClusterReconciler {
	r := &ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	r.heartbeatTime()
	go func() {
		for {
			_, ok := <-mgr.Elected()
			if !ok {
				break
			}
			time.Sleep(2 * time.Second)
		}
		log.Info("start cluster registration")
		registerClusterInstance.isClusterExist().
			registerCluster().isClusterApproveOrNot().
			syncAuthData(mgr).heartbeat()
	}()
	return r
}

type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if registerClusterInstance.currentCluster.Name == "" {
		return ctrl.Result{RequeueAfter: 120 * time.Second}, nil
	}
	node := &v1.Node{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name}, node)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infow("Node has been deleted", "Node", req)
		} else {
			return ctrl.Result{}, err
		}
	}
	var g cluster.Cluster
	if err := registerClusterInstance.cli.Get(registerClusterInstance.currentCluster.Name, nil, &g); err != nil {
		return ctrl.Result{}, err
	}
	lock.Lock()
	registerClusterInstance.currentCluster = &g
	lock.Unlock()
	nodes := r.getNodeList(context.Background())
	r.kubeVersion(nodes)
	r.clusterNodeCount(nodes)
	r.clusterResouce(nodes)
	r.heartbeatTime()
	if err := registerClusterInstance.cli.Update(fmt.Sprintf("%d", registerClusterInstance.currentCluster.ID),
		registerClusterInstance.currentCluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldStatus := e.ObjectOld.(*v1.Node).Status
				newStatus := e.ObjectNew.(*v1.Node).Status
				if !cmp.Equal(oldStatus.Allocatable, newStatus.Allocatable) ||
					!cmp.Equal(oldStatus.Capacity, newStatus.Capacity) ||
					oldStatus.NodeInfo.KubeletVersion != newStatus.NodeInfo.KubeletVersion {
					return true
				}
				change := false
				for i, v := range oldStatus.Conditions {
					if v.Status != newStatus.Conditions[i].Status {
						change = true
					}
				}
				return change
			},
		})).
		Complete(r)
}

func (r *ClusterReconciler) getNodeList(ctx context.Context) []v1.Node {
	nodeList := v1.NodeList{}
	if err := r.List(ctx, &nodeList); err != nil {
		log.Error(err, "get node list with error")
	}
	return nodeList.Items
}

func (r *ClusterReconciler) kubeVersion(nodes []v1.Node) {
	lock.Lock()
	defer lock.Unlock()
	if len(nodes) > 0 {
		registerClusterInstance.currentCluster.KubernetesVersion = nodes[0].Status.NodeInfo.KubeletVersion
		return
	}
	registerClusterInstance.currentCluster.KubernetesVersion = "unknown"
}

func (r *ClusterReconciler) clusterNodeCount(nodes []v1.Node) {
	ready, notReady := 0, 0
	for _, node := range nodes {
		l := len(node.Status.Conditions)
		for i, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
				ready++
				break
			}
			if i == l-1 {
				notReady++
			}
		}
	}
	lock.Lock()
	registerClusterInstance.currentCluster.ReadyNodeCount = ready
	registerClusterInstance.currentCluster.UnhealthNodeCount = notReady
	lock.Unlock()
}

func (r *ClusterReconciler) clusterResouce(nodes []v1.Node) {
	var capacityCpu, capacityMem, allocatableCpu, allocatableMem resource.Quantity
	for _, node := range nodes {
		capacityCpu.Add(*node.Status.Capacity.Cpu())
		capacityMem.Add(*node.Status.Capacity.Memory())
		allocatableCpu.Add(*node.Status.Allocatable.Cpu())
		allocatableMem.Add(*node.Status.Allocatable.Memory())
	}
	lock.Lock()
	registerClusterInstance.currentCluster.AllocatableCPU = allocatableCpu.String()
	registerClusterInstance.currentCluster.CapacityCPU = capacityCpu.String()
	registerClusterInstance.currentCluster.AllocatableMemory = allocatableMem.String()
	registerClusterInstance.currentCluster.CapacityMemory = capacityMem.String()
	lock.Unlock()
}

func (r *ClusterReconciler) heartbeatTime() {
	lock.Lock()
	registerClusterInstance.currentCluster.LastKeepAliveTime = time.Now().Local()
	lock.Unlock()
}
