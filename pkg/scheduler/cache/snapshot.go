package cache

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Snapshot is a snapshot of cache NodeInfo and NodeTree order. The scheduler takes a
// snapshot at the beginning of each scheduling cycle and uses it for its operations in that cycle.
type Snapshot struct {
	// nodeInfoMap a map of node name to a snapshot of its NodeInfo.
	nodeInfoMap map[string]*framework.NodeInfo
	// nodeInfoList is the list of nodes as ordered in the cache's nodeTree.
	nodeInfoList []*framework.NodeInfo
	// havePodsWithAffinityNodeInfoList is the list of nodes with at least one pod declaring affinity terms.
	havePodsWithAffinityNodeInfoList []*framework.NodeInfo
	// havePodsWithRequiredAntiAffinityNodeInfoList is the list of nodes with at least one pod declaring
	// required anti-affinity terms.
	havePodsWithRequiredAntiAffinityNodeInfoList []*framework.NodeInfo
	generation                                   int64
}

// var _ framework.SharedLister = &Snapshot{}

// NewEmptySnapshot initializes a Snapshot struct and returns it.
func NewEmptySnapshot() *Snapshot {
	return &Snapshot{
		nodeInfoMap: make(map[string]*framework.NodeInfo),
	}
}

// NewSnapshot initializes a Snapshot struct and returns it.
func NewSnapshot(pods []*v1.Pod, nodes []*v1.Node) *Snapshot {
	nodeInfoMap := createNodeInfoMap(pods, nodes)
	nodeInfoList := make([]*framework.NodeInfo, 0, len(nodeInfoMap))
	havePodsWithAffinityNodeInfoList := make([]*framework.NodeInfo, 0, len(nodeInfoMap))
	havePodsWithRequiredAntiAffinityNodeInfoList := make([]*framework.NodeInfo, 0, len(nodeInfoMap))
	for _, v := range nodeInfoMap {
		nodeInfoList = append(nodeInfoList, v)
		if len(v.PodsWithAffinity) > 0 {
			havePodsWithAffinityNodeInfoList = append(havePodsWithAffinityNodeInfoList, v)
		}
		if len(v.PodsWithRequiredAntiAffinity) > 0 {
			havePodsWithRequiredAntiAffinityNodeInfoList = append(havePodsWithRequiredAntiAffinityNodeInfoList, v)
		}
	}

	s := NewEmptySnapshot()
	s.nodeInfoMap = nodeInfoMap
	s.nodeInfoList = nodeInfoList
	s.havePodsWithAffinityNodeInfoList = havePodsWithAffinityNodeInfoList
	s.havePodsWithRequiredAntiAffinityNodeInfoList = havePodsWithRequiredAntiAffinityNodeInfoList

	return s
}

// createNodeInfoMap obtains a list of pods and pivots that list into a map
// where the keys are node names and the values are the aggregated information
// for that node.
func createNodeInfoMap(pods []*v1.Pod, nodes []*v1.Node) map[string]*framework.NodeInfo {
	nodeNameToInfo := make(map[string]*framework.NodeInfo)
	for _, pod := range pods {
		nodeName := pod.Spec.NodeName
		if _, ok := nodeNameToInfo[nodeName]; !ok {
			nodeNameToInfo[nodeName] = framework.NewNodeInfo()
		}
		nodeNameToInfo[nodeName].AddPod(pod)
	}

	for _, node := range nodes {
		if _, ok := nodeNameToInfo[node.Name]; !ok {
			nodeNameToInfo[node.Name] = framework.NewNodeInfo()
		}
		nodeInfo := nodeNameToInfo[node.Name]
		nodeInfo.SetNode(node)
	}
	return nodeNameToInfo
}

// NodeInfos returns a NodeInfoLister.
func (s *Snapshot) NodeInfos() framework.NodeInfoLister {
	return s
}

// NumNodes returns the number of nodes in the snapshot.
func (s *Snapshot) NumNodes() int {
	return len(s.nodeInfoList)
}

// List returns the list of nodes in the snapshot.
func (s *Snapshot) List() ([]*framework.NodeInfo, error) {
	return s.nodeInfoList, nil
}

// HavePodsWithAffinityList returns the list of nodes with at least one pod with inter-pod affinity
func (s *Snapshot) HavePodsWithAffinityList() ([]*framework.NodeInfo, error) {
	return s.havePodsWithAffinityNodeInfoList, nil
}

// HavePodsWithRequiredAntiAffinityList returns the list of nodes with at least one pod with
// required inter-pod anti-affinity
func (s *Snapshot) HavePodsWithRequiredAntiAffinityList() ([]*framework.NodeInfo, error) {
	return s.havePodsWithRequiredAntiAffinityNodeInfoList, nil
}

// Get returns the NodeInfo of the given node name.
func (s *Snapshot) Get(nodeName string) (*framework.NodeInfo, error) {
	if v, ok := s.nodeInfoMap[nodeName]; ok && v.Node() != nil {
		return v, nil
	}
	return nil, fmt.Errorf("nodeinfo not found for node name %q", nodeName)
}
