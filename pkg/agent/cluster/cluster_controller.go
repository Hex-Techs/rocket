package cluster

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
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
		klog.Info("start cluster registration")
		registerInstance.isClusterExist().
			registerCluster(mgr).isClusterApproveOrNot(mgr).
			syncAuthData(mgr).heartbeat(mgr)
	}()
	return r
}

type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if registerInstance.currentCluster.Name == "" {
		return ctrl.Result{RequeueAfter: 120 * time.Second}, nil
	}
	node := &v1.Node{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name}, node)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("Node '%s' has been deleted.", req)
		} else {
			return ctrl.Result{}, err
		}
	}
	g, err := registerInstance.cli.RocketV1alpha1().Clusters().Get(context.TODO(),
		registerInstance.currentCluster.Name, metav1.GetOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	lock.Lock()
	registerInstance.currentCluster = g
	lock.Unlock()
	nodes := r.getNodeList(context.TODO())
	r.kubeVersion(nodes)
	r.clusterNodeCount(nodes)
	r.clusterResouce(nodes)
	r.heartbeatTime()
	_, err = registerInstance.cli.RocketV1alpha1().Clusters().UpdateStatus(context.TODO(), registerInstance.currentCluster, metav1.UpdateOptions{})
	if err != nil {
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
		klog.Errorf("get node list with error: %v", err)
	}
	return nodeList.Items
}

func (r *ClusterReconciler) kubeVersion(nodes []v1.Node) {
	lock.Lock()
	defer lock.Unlock()
	if len(nodes) > 0 {
		registerInstance.currentCluster.Status.KubernetesVersion = nodes[0].Status.NodeInfo.KubeletVersion
		return
	}
	registerInstance.currentCluster.Status.KubernetesVersion = "unknown"
}

func (r *ClusterReconciler) clusterNodeCount(nodes []v1.Node) {
	ready, notReady := 0, 0
	for _, node := range nodes {
		if registerInstance.nodere.MatchString(node.Name) {
			continue
		}
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
	registerInstance.currentCluster.Status.ReadyNodeCount = &ready
	registerInstance.currentCluster.Status.UnhealthNodeCount = &notReady
	lock.Unlock()
}

func (r *ClusterReconciler) clusterResouce(nodes []v1.Node) {
	var capacityCpu, capacityMem, allocatableCpu, allocatableMem resource.Quantity
	Capacity, Allocatable := make(map[v1.ResourceName]resource.Quantity), make(map[v1.ResourceName]resource.Quantity)
	for _, node := range nodes {
		if registerInstance.nodere.MatchString(node.Name) {
			continue
		}
		capacityCpu.Add(*node.Status.Capacity.Cpu())
		capacityMem.Add(*node.Status.Capacity.Memory())
		allocatableCpu.Add(*node.Status.Allocatable.Cpu())
		allocatableMem.Add(*node.Status.Allocatable.Memory())
	}

	Capacity[v1.ResourceCPU] = capacityCpu
	Capacity[v1.ResourceMemory] = capacityMem
	Allocatable[v1.ResourceCPU] = allocatableCpu
	Allocatable[v1.ResourceMemory] = allocatableMem
	lock.Lock()
	registerInstance.currentCluster.Status.Allocatable = Allocatable
	registerInstance.currentCluster.Status.Capacity = Capacity
	lock.Unlock()
}

func (r *ClusterReconciler) heartbeatTime() {
	lock.Lock()
	registerInstance.currentCluster.Status.LastKeepAliveTime = metav1.Now()
	lock.Unlock()
}
