package cache

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Cache collects pods' information and provides node-level aggregated information.
// It's intended for generic scheduler to do efficient lookup.
// Cache's operations are pod centric. It does incremental updates based on pod events.
// Pod events are sent via network. We don't have guaranteed delivery of all events:
// We use Reflector to list and watch from remote.
// Reflector might be slow and do a relist, which would lead to missing events.
//
// State Machine of a pod's events in scheduler's cache:
//
//	+-------------------------------------------+  +----+
//	|                            Add            |  |    |
//	|                                           |  |    | Update
//	+      Assume                Add            v  v    |
//
// Initial +--------> Assumed +------------+---> Added <--+
//
//	^                +   +               |       +
//	|                |   |               |       |
//	|                |   |           Add |       | Remove
//	|                |   |               |       |
//	|                |   |               +       |
//	+----------------+   +-----------> Expired   +----> Deleted
//	      Forget             Expire
//
// Note that an assumed pod can expire, because if we haven't received Add event notifying us
// for a while, there might be some problems and we shouldn't keep the pod in cache anymore.
//
// Note that "Initial", "Expired", and "Deleted" pods do not actually exist in cache.
// Based on existing use cases, we are making the following assumptions:
//   - No pod would be assumed twice
//   - A pod could be added without going through scheduler. In this case, we will see Add but not Assume event.
//   - If a pod wasn't added, it wouldn't be removed or updated.
//   - Both "Expired" and "Deleted" are valid end states. In case of some problems, e.g. network issue,
//     a pod might have changed its state (e.g. added and deleted) without delivering notification to the cache.
type Cache interface {
	// NodeCount returns the number of nodes in the cache.
	// DO NOT use outside of tests.
	NodeCount() int

	// PodCount returns the number of pods in the cache (including those from deleted nodes).
	// DO NOT use outside of tests.
	PodCount() (int, error)

	// AddPod either confirms a pod if it's assumed, or adds it back if it's expired.
	// If added back, the pod's information would be added again.
	AddPod(pod *v1.Pod) error

	// UpdatePod removes oldPod's information and adds newPod's information.
	UpdatePod(oldPod, newPod *v1.Pod) error

	// RemovePod removes a pod. The pod's information would be subtracted from assigned node.
	RemovePod(pod *v1.Pod) error

	// GetPod returns the pod from the cache with the same namespace and the
	// same name of the specified pod.
	GetPod(pod *v1.Pod) (*v1.Pod, error)

	// AddNode adds overall information about node.
	// It returns a clone of added NodeInfo object.
	AddNode(node *v1.Node) *framework.NodeInfo

	// UpdateNode updates overall information about node.
	// It returns a clone of updated NodeInfo object.
	UpdateNode(oldNode, newNode *v1.Node) *framework.NodeInfo

	// RemoveNode removes overall information about node.
	RemoveNode(node *v1.Node) error

	// UpdateSnapshot updates the passed infoSnapshot to the current contents of Cache.
	// The node info contains aggregated information of pods scheduled (including assumed to be)
	// on this node.
	// The snapshot only includes Nodes that are not deleted at the time this function is called.
	// nodeinfo.Node() is guaranteed to be not nil for all the nodes in the snapshot.
	UpdateSnapshot(nodeSnapshot *Snapshot) error

	// Dump produces a dump of the current cache.
	Dump() *Dump
}

// Dump is a dump of the cache state.
type Dump struct {
	AssumedPods sets.String
	Nodes       map[string]*framework.NodeInfo
}
