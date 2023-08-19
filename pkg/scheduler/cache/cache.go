package cache

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hex-techs/rocket/pkg/utils/tools"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

var (
	cleanAssumedPeriod = 1 * time.Second
)

// New returns a Cache implementation.
// It automatically starts a go routine that manages expiration of assumed pods.
// "ttl" is how long the assumed pod will get expired.
// "stop" is the channel that would close the background goroutine.
func New(ttl time.Duration, stop <-chan struct{}) Cache {
	cache := newCache(ttl, cleanAssumedPeriod, stop)
	cache.run()
	return cache
}

// nodeInfoListItem holds a NodeInfo pointer and acts as an item in a doubly
// linked list. When a NodeInfo is updated, it goes to the head of the list.
// The items closer to the head are the most recently updated items.
type nodeInfoListItem struct {
	info *framework.NodeInfo
	next *nodeInfoListItem
	prev *nodeInfoListItem
}

type cacheImpl struct {
	stop   <-chan struct{}
	ttl    time.Duration
	period time.Duration

	// This mutex guards all fields within this cache struct.
	mu sync.RWMutex
	// a set of assumed pod keys.
	// The key could further be used to get an entry in podStates.
	assumedPods sets.String
	// a map from pod key to podState.
	podStates map[string]*podState
	nodes     map[string]*nodeInfoListItem
	// headNode points to the most recently updated NodeInfo in "nodes". It is the
	// head of the linked list.
	headNode *nodeInfoListItem
	nodeTree *nodeTree
}

type podState struct {
	pod *v1.Pod
	// Used by assumedPod to determinate expiration.
	deadline *time.Time
	// Used to block cache from expiring assumedPod if binding still runs
	bindingFinished bool
}

func newCache(ttl, period time.Duration, stop <-chan struct{}) *cacheImpl {
	return &cacheImpl{
		ttl:    ttl,
		period: period,
		stop:   stop,

		nodes:       make(map[string]*nodeInfoListItem),
		nodeTree:    newNodeTree(nil),
		assumedPods: make(sets.String),
		podStates:   make(map[string]*podState),
	}
}

// newNodeInfoListItem initializes a new nodeInfoListItem.
func newNodeInfoListItem(ni *framework.NodeInfo) *nodeInfoListItem {
	return &nodeInfoListItem{
		info: ni,
	}
}

// moveNodeInfoToHead moves a NodeInfo to the head of "cache.nodes" doubly
// linked list. The head is the most recently updated NodeInfo.
// We assume cache lock is already acquired.
func (cache *cacheImpl) moveNodeInfoToHead(name string) {
	ni, ok := cache.nodes[name]
	if !ok {
		log.Error(nil, "No node info with given name found in the cache", "node", tools.KRef("", name))
		return
	}
	// if the node info list item is already at the head, we are done.
	if ni == cache.headNode {
		return
	}

	if ni.prev != nil {
		ni.prev.next = ni.next
	}
	if ni.next != nil {
		ni.next.prev = ni.prev
	}
	if cache.headNode != nil {
		cache.headNode.prev = ni
	}
	ni.next = cache.headNode
	ni.prev = nil
	cache.headNode = ni
}

// removeNodeInfoFromList removes a NodeInfo from the "cache.nodes" doubly
// linked list.
// We assume cache lock is already acquired.
func (cache *cacheImpl) removeNodeInfoFromList(name string) {
	ni, ok := cache.nodes[name]
	if !ok {
		log.Error(nil, "No node info with given name found in the cache", "node", tools.KRef("", name))
		return
	}

	if ni.prev != nil {
		ni.prev.next = ni.next
	}
	if ni.next != nil {
		ni.next.prev = ni.prev
	}
	// if the removed item was at the head, we must update the head.
	if ni == cache.headNode {
		cache.headNode = ni.next
	}
	delete(cache.nodes, name)
}

// Dump produces a dump of the current scheduler cache. This is used for
// debugging purposes only and shouldn't be confused with UpdateSnapshot
// function.
// This method is expensive, and should be only used in non-critical path.
func (cache *cacheImpl) Dump() *Dump {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	nodes := make(map[string]*framework.NodeInfo, len(cache.nodes))
	for k, v := range cache.nodes {
		nodes[k] = v.info.Clone()
	}

	return &Dump{
		Nodes:       nodes,
		AssumedPods: cache.assumedPods.Union(nil),
	}
}

// UpdateSnapshot takes a snapshot of cached NodeInfo map. This is called at
// beginning of every scheduling cycle.
// The snapshot only includes Nodes that are not deleted at the time this function is called.
// nodeinfo.Node() is guaranteed to be not nil for all the nodes in the snapshot.
// This function tracks generation number of NodeInfo and updates only the
// entries of an existing snapshot that have changed after the snapshot was taken.
func (cache *cacheImpl) UpdateSnapshot(nodeSnapshot *Snapshot) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Get the last generation of the snapshot.
	snapshotGeneration := nodeSnapshot.generation

	// NodeInfoList and HavePodsWithAffinityNodeInfoList must be re-created if a node was added
	// or removed from the cache.
	updateAllLists := false
	// HavePodsWithAffinityNodeInfoList must be re-created if a node changed its
	// status from having pods with affinity to NOT having pods with affinity or the other
	// way around.
	updateNodesHavePodsWithAffinity := false
	// HavePodsWithRequiredAntiAffinityNodeInfoList must be re-created if a node changed its
	// status from having pods with required anti-affinity to NOT having pods with required
	// anti-affinity or the other way around.
	updateNodesHavePodsWithRequiredAntiAffinity := false

	// Start from the head of the NodeInfo doubly linked list and update snapshot
	// of NodeInfos updated after the last snapshot.
	for node := cache.headNode; node != nil; node = node.next {
		if node.info.Generation <= snapshotGeneration {
			// all the nodes are updated before the existing snapshot. We are done.
			break
		}
		if np := node.info.Node(); np != nil {
			existing, ok := nodeSnapshot.nodeInfoMap[np.Name]
			if !ok {
				updateAllLists = true
				existing = &framework.NodeInfo{}
				nodeSnapshot.nodeInfoMap[np.Name] = existing
			}
			clone := node.info.Clone()
			// We track nodes that have pods with affinity, here we check if this node changed its
			// status from having pods with affinity to NOT having pods with affinity or the other
			// way around.
			if (len(existing.PodsWithAffinity) > 0) != (len(clone.PodsWithAffinity) > 0) {
				updateNodesHavePodsWithAffinity = true
			}
			if (len(existing.PodsWithRequiredAntiAffinity) > 0) != (len(clone.PodsWithRequiredAntiAffinity) > 0) {
				updateNodesHavePodsWithRequiredAntiAffinity = true
			}
			// We need to preserve the original pointer of the NodeInfo struct since it
			// is used in the NodeInfoList, which we may not update.
			*existing = *clone
		}
	}
	// Update the snapshot generation with the latest NodeInfo generation.
	if cache.headNode != nil {
		nodeSnapshot.generation = cache.headNode.info.Generation
	}

	// Comparing to pods in nodeTree.
	// Deleted nodes get removed from the tree, but they might remain in the nodes map
	// if they still have non-deleted Pods.
	if len(nodeSnapshot.nodeInfoMap) > cache.nodeTree.numNodes {
		cache.removeDeletedNodesFromSnapshot(nodeSnapshot)
		updateAllLists = true
	}

	if updateAllLists || updateNodesHavePodsWithAffinity || updateNodesHavePodsWithRequiredAntiAffinity {
		cache.updateNodeInfoSnapshotList(nodeSnapshot, updateAllLists)
	}

	if len(nodeSnapshot.nodeInfoList) != cache.nodeTree.numNodes {
		errMsg := fmt.Sprintf("snapshot state is not consistent, length of NodeInfoList=%v not equal to length of nodes in tree=%v "+
			", length of NodeInfoMap=%v, length of nodes in cache=%v"+
			", trying to recover",
			len(nodeSnapshot.nodeInfoList), cache.nodeTree.numNodes,
			len(nodeSnapshot.nodeInfoMap), len(cache.nodes))
		log.Error(nil, errMsg)
		// We will try to recover by re-creating the lists for the next scheduling cycle, but still return an
		// error to surface the problem, the error will likely cause a failure to the current scheduling cycle.
		cache.updateNodeInfoSnapshotList(nodeSnapshot, true)
		return fmt.Errorf(errMsg)
	}

	return nil
}

func (cache *cacheImpl) updateNodeInfoSnapshotList(snapshot *Snapshot, updateAll bool) {
	snapshot.havePodsWithAffinityNodeInfoList = make([]*framework.NodeInfo, 0, cache.nodeTree.numNodes)
	snapshot.havePodsWithRequiredAntiAffinityNodeInfoList = make([]*framework.NodeInfo, 0, cache.nodeTree.numNodes)
	if updateAll {
		// Take a snapshot of the nodes order in the tree
		snapshot.nodeInfoList = make([]*framework.NodeInfo, 0, cache.nodeTree.numNodes)
		nodesList, err := cache.nodeTree.list()
		if err != nil {
			log.Error(err, "Error occurred while retrieving the list of names of the nodes from node tree")
		}
		for _, nodeName := range nodesList {
			if nodeInfo := snapshot.nodeInfoMap[nodeName]; nodeInfo != nil {
				snapshot.nodeInfoList = append(snapshot.nodeInfoList, nodeInfo)
				if len(nodeInfo.PodsWithAffinity) > 0 {
					snapshot.havePodsWithAffinityNodeInfoList = append(snapshot.havePodsWithAffinityNodeInfoList, nodeInfo)
				}
				if len(nodeInfo.PodsWithRequiredAntiAffinity) > 0 {
					snapshot.havePodsWithRequiredAntiAffinityNodeInfoList = append(snapshot.havePodsWithRequiredAntiAffinityNodeInfoList, nodeInfo)
				}
			} else {
				log.Error(nil, "Node exists in nodeTree but not in NodeInfoMap, this should not happen", "node", tools.KRef("", nodeName))
			}
		}
	} else {
		for _, nodeInfo := range snapshot.nodeInfoList {
			if len(nodeInfo.PodsWithAffinity) > 0 {
				snapshot.havePodsWithAffinityNodeInfoList = append(snapshot.havePodsWithAffinityNodeInfoList, nodeInfo)
			}
			if len(nodeInfo.PodsWithRequiredAntiAffinity) > 0 {
				snapshot.havePodsWithRequiredAntiAffinityNodeInfoList = append(snapshot.havePodsWithRequiredAntiAffinityNodeInfoList, nodeInfo)
			}
		}
	}
}

// If certain nodes were deleted after the last snapshot was taken, we should remove them from the snapshot.
func (cache *cacheImpl) removeDeletedNodesFromSnapshot(snapshot *Snapshot) {
	toDelete := len(snapshot.nodeInfoMap) - cache.nodeTree.numNodes
	for name := range snapshot.nodeInfoMap {
		if toDelete <= 0 {
			break
		}
		if n, ok := cache.nodes[name]; !ok || n.info.Node() == nil {
			delete(snapshot.nodeInfoMap, name)
			toDelete--
		}
	}
}

// NodeCount returns the number of nodes in the cache.
// DO NOT use outside of tests.
func (cache *cacheImpl) NodeCount() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return len(cache.nodes)
}

// PodCount returns the number of pods in the cache (including those from deleted nodes).
// DO NOT use outside of tests.
func (cache *cacheImpl) PodCount() (int, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	// podFilter is expected to return true for most or all of the pods. We
	// can avoid expensive array growth without wasting too much memory by
	// pre-allocating capacity.
	count := 0
	for _, n := range cache.nodes {
		count += len(n.info.Pods)
	}
	return count, nil
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) addPod(pod *v1.Pod, assumePod bool) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}
	n, ok := cache.nodes[pod.Spec.NodeName]
	if !ok {
		n = newNodeInfoListItem(framework.NewNodeInfo())
		cache.nodes[pod.Spec.NodeName] = n
	}
	n.info.AddPod(pod)
	cache.moveNodeInfoToHead(pod.Spec.NodeName)
	ps := &podState{
		pod: pod,
	}
	cache.podStates[key] = ps
	if assumePod {
		cache.assumedPods.Insert(key)
	}
	return nil
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) updatePod(oldPod, newPod *v1.Pod) error {
	if err := cache.removePod(oldPod); err != nil {
		return err
	}
	return cache.addPod(newPod, false)
}

// Assumes that lock is already acquired.
// Removes a pod from the cached node info. If the node information was already
// removed and there are no more pods left in the node, cleans up the node from
// the cache.
func (cache *cacheImpl) removePod(pod *v1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	n, ok := cache.nodes[pod.Spec.NodeName]
	if !ok {
		log.Error(nil, "Node not found when trying to remove pod", "node", tools.KRef("", pod.Spec.NodeName), "pod", tools.KObj(pod))
	} else {
		if err := n.info.RemovePod(pod); err != nil {
			return err
		}
		if len(n.info.Pods) == 0 && n.info.Node() == nil {
			cache.removeNodeInfoFromList(pod.Spec.NodeName)
		} else {
			cache.moveNodeInfoToHead(pod.Spec.NodeName)
		}
	}

	delete(cache.podStates, key)
	delete(cache.assumedPods, key)
	return nil
}

func (cache *cacheImpl) AddPod(pod *v1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.podStates[key]
	switch {
	case ok && cache.assumedPods.Has(key):
		if currState.pod.Spec.NodeName != pod.Spec.NodeName {
			// The pod was added to a different node than it was assumed to.
			log.V(0).Info("Pod was added to a different node than it was assumed", "pod", tools.KObj(pod), "assumedNode", tools.KRef("", pod.Spec.NodeName), "currentNode", tools.KRef("", currState.pod.Spec.NodeName))
			if err = cache.updatePod(currState.pod, pod); err != nil {
				log.Error(err, "Error occurred while updating pod")
			}
		} else {
			delete(cache.assumedPods, key)
			cache.podStates[key].deadline = nil
			cache.podStates[key].pod = pod
		}
	case !ok:
		// Pod was expired. We should add it back.
		if err = cache.addPod(pod, false); err != nil {
			log.Error(err, "Error occurred while adding pod")
		}
	default:
		return fmt.Errorf("pod %v was already in added state", key)
	}
	return nil
}

func (cache *cacheImpl) UpdatePod(oldPod, newPod *v1.Pod) error {
	key, err := framework.GetPodKey(oldPod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.podStates[key]
	// An assumed pod won't have Update/Remove event. It needs to have Add event
	// before Update event, in which case the state would change from Assumed to Added.
	if ok && !cache.assumedPods.Has(key) {
		if currState.pod.Spec.NodeName != newPod.Spec.NodeName {
			log.Error(nil, "Pod updated on a different node than previously added to", "pod", tools.KObj(oldPod))
			log.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
			os.Exit(1)
		}
		return cache.updatePod(oldPod, newPod)
	}
	return fmt.Errorf("pod %v is not added to scheduler cache, so cannot be updated", key)
}

func (cache *cacheImpl) RemovePod(pod *v1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.podStates[key]
	if !ok {
		return fmt.Errorf("pod %v is not found in scheduler cache, so cannot be removed from it", key)
	}
	if currState.pod.Spec.NodeName != pod.Spec.NodeName {
		log.Error(nil, "Pod was added to a different node than it was assumed", "pod", tools.KObj(pod), "assumedNode", tools.KRef("", pod.Spec.NodeName), "currentNode", tools.KRef("", currState.pod.Spec.NodeName))
		if pod.Spec.NodeName != "" {
			// An empty NodeName is possible when the scheduler misses a Delete
			// event and it gets the last known state from the informer cache.
			log.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
			os.Exit(1)
		}
	}
	return cache.removePod(currState.pod)
}

// GetPod might return a pod for which its node has already been deleted from
// the main cache. This is useful to properly process pod update events.
func (cache *cacheImpl) GetPod(pod *v1.Pod) (*v1.Pod, error) {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return nil, err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	podState, ok := cache.podStates[key]
	if !ok {
		return nil, fmt.Errorf("pod %v does not exist in scheduler cache", key)
	}

	return podState.pod, nil
}

func (cache *cacheImpl) AddNode(node *v1.Node) *framework.NodeInfo {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.nodes[node.Name]
	if !ok {
		n = newNodeInfoListItem(framework.NewNodeInfo())
		cache.nodes[node.Name] = n
	}
	cache.moveNodeInfoToHead(node.Name)

	cache.nodeTree.addNode(node)
	n.info.SetNode(node)
	return n.info.Clone()
}

func (cache *cacheImpl) UpdateNode(oldNode, newNode *v1.Node) *framework.NodeInfo {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.nodes[newNode.Name]
	if !ok {
		n = newNodeInfoListItem(framework.NewNodeInfo())
		cache.nodes[newNode.Name] = n
		cache.nodeTree.addNode(newNode)
	}
	cache.moveNodeInfoToHead(newNode.Name)

	cache.nodeTree.updateNode(oldNode, newNode)
	n.info.SetNode(newNode)
	return n.info.Clone()
}

// RemoveNode removes a node from the cache's tree.
// The node might still have pods because their deletion events didn't arrive
// yet. Those pods are considered removed from the cache, being the node tree
// the source of truth.
// However, we keep a ghost node with the list of pods until all pod deletion
// events have arrived. A ghost node is skipped from snapshots.
func (cache *cacheImpl) RemoveNode(node *v1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.nodes[node.Name]
	if !ok {
		return fmt.Errorf("node %v is not found", node.Name)
	}
	n.info.RemoveNode()
	// We remove NodeInfo for this node only if there aren't any pods on this node.
	// We can't do it unconditionally, because notifications about pods are delivered
	// in a different watch, and thus can potentially be observed later, even though
	// they happened before node removal.
	if len(n.info.Pods) == 0 {
		cache.removeNodeInfoFromList(node.Name)
	} else {
		cache.moveNodeInfoToHead(node.Name)
	}
	if err := cache.nodeTree.removeNode(node); err != nil {
		return err
	}
	return nil
}

func (cache *cacheImpl) run() {
	go wait.Until(cache.cleanupExpiredAssumedPods, cache.period, cache.stop)
}

func (cache *cacheImpl) cleanupExpiredAssumedPods() {
	cache.cleanupAssumedPods(time.Now())
}

// cleanupAssumedPods exists for making test deterministic by taking time as input argument.
func (cache *cacheImpl) cleanupAssumedPods(now time.Time) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// The size of assumedPods should be small
	for key := range cache.assumedPods {
		ps, ok := cache.podStates[key]
		if !ok {
			log.Error(nil, "Key found in assumed set but not in podStates, potentially a logical error")
			os.Exit(1)
		}
		if !ps.bindingFinished {
			log.V(4).Info("Could not expire cache for pod as binding is still in progress",
				"pod", tools.KObj(ps.pod))
			continue
		}
		if now.After(*ps.deadline) {
			log.V(0).Info("Pod expired", "pod", tools.KObj(ps.pod))
			if err := cache.removePod(ps.pod); err != nil {
				log.Error(err, "ExpirePod failed", "pod", tools.KObj(ps.pod))
			}
		}
	}
}
