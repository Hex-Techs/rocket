package schedule

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	schedutil "k8s.io/kubernetes/pkg/scheduler/util"
)

// resourceToWeightMap 包含资源名称和权重.
type resourceToWeightMap map[v1.ResourceName]int64

// resourceToValueMap 资源名称和数量.
type resourceToValueMap map[v1.ResourceName]int64

// 节点资源过滤，计算节点可以分配的pod数量
func resourceFilter(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) (int64, error) {
	node := nodeInfo.Node()
	if node == nil {
		return 0, errors.New("node not found")
	}
	cr := makeResourceSpec(pod.Spec.Containers)
	rtwm := resourcesToWeightMap(cr)
	if rtwm == nil {
		return 0, errors.New("resources not found")
	}
	requested := make(resourceToValueMap)
	allocatable := make(resourceToValueMap)
	request := make(resourceToValueMap)
	for resource := range rtwm {
		alloc, req := calculateResourceAllocatableRequest(nodeInfo, resource)
		if alloc != 0 {
			// Only fill the extended resource entry when it's non-zero.
			allocatable[resource], requested[resource] = alloc, req
		}
		v := calculatePodResourceRequest(pod, resource)
		if v != 0 {
			request[resource] = v
		}
	}
	klog.V(4).Infof("node: %v request: %+v requested: %+v allocatable: %+v\n", node.Name, request, requested, allocatable)
	replicas := calculateNodeCanAllocatePodReplicas(request, requested, allocatable)
	klog.V(4).Infof("%v: According to Allocation, Replicas: (%d)", node.Name, replicas)
	return replicas, nil
}

// 计算节点可以分配的pod数量
func calculateNodeCanAllocatePodReplicas(request, requested, allocable resourceToValueMap) int64 {
	var minReplicas []int64
	for resource := range request {
		requestValue := request[resource]
		requestedValue := requested[resource]
		allocableValue := allocable[resource]
		if allocableValue < requestValue {
			klog.V(4).Infof("request value %d is larger than allocable value %d for resource %s", requestValue, allocableValue, resource)
			return 0
		}
		if allocableValue < requestedValue {
			klog.V(4).Infof("calculate error requested value %d is larger than allocable value %d for resource %s", requestedValue, allocableValue, resource)
			return 0
		}
		minReplicas = append(minReplicas, (allocableValue-requestedValue)/requestValue)
	}
	if len(minReplicas) == 0 {
		return 0
	}
	min := minReplicas[0]
	for _, v := range minReplicas {
		if v < min {
			min = v
		}
	}
	return min
}

// calculatePodResourceRequest 返回非0请求总数. rocket没有使用pod overhead（开销）特性
// podResourceRequest = max(sum(podSpec.Containers), podSpec.InitContainers) + overHead
func calculatePodResourceRequest(pod *v1.Pod, resource v1.ResourceName) int64 {
	var podRequest int64
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		value := schedutil.GetRequestForResource(resource, &container.Resources.Requests, false)
		podRequest += value
	}
	for i := range pod.Spec.InitContainers {
		initContainer := &pod.Spec.InitContainers[i]
		value := schedutil.GetRequestForResource(resource, &initContainer.Resources.Requests, false)
		if podRequest < value {
			podRequest = value
		}
	}
	// 关于pod开销的计算，rocket没有使用pod overhead（开销）特性
	if pod.Spec.Overhead != nil {
		if quantity, found := pod.Spec.Overhead[resource]; found {
			podRequest += quantity.Value()
		}
	}
	return podRequest
}

// calculateResourceAllocatableRequest 返回两个参数:
// - 1st param: 节点上可分配资源的数量.
// - 2nd param: 节点上请求的资源的聚合数量，不包括待分配pod.
// Note: 如果资源是扩展资源，并且pod没有使用这个资源, 那么默认返回(0, 0).
func calculateResourceAllocatableRequest(nodeInfo *framework.NodeInfo, resource v1.ResourceName) (int64, int64) {
	// requested := nodeInfo.NonZeroRequested
	requested := nodeInfo.Requested
	switch resource {
	case v1.ResourceCPU:
		return nodeInfo.Allocatable.MilliCPU, requested.MilliCPU
	case v1.ResourceMemory:
		return nodeInfo.Allocatable.Memory, requested.Memory
	case v1.ResourceEphemeralStorage:
		return nodeInfo.Allocatable.EphemeralStorage, nodeInfo.Requested.EphemeralStorage
	default:
		if _, exists := nodeInfo.Allocatable.ScalarResources[resource]; exists {
			return nodeInfo.Allocatable.ScalarResources[resource], nodeInfo.Requested.ScalarResources[resource]
		}
	}
	if klog.V(10).Enabled() {
		klog.V(4).InfoS("Requested resource is omitted for node score calculation", "resourceName", resource)
	}
	return 0, 0
}

// resourcesToWeightMap make weightmap from resources spec
func resourcesToWeightMap(resources []config.ResourceSpec) resourceToWeightMap {
	resourceToWeightMap := make(resourceToWeightMap)
	for _, resource := range resources {
		resourceToWeightMap[v1.ResourceName(resource.Name)] = resource.Weight
	}
	return resourceToWeightMap
}
func isExist(x string, y []config.ResourceSpec) bool {
	for _, r := range y {
		if r.Name == x {
			return true
		}
	}
	return false
}
func makeResourceSpec(containers []v1.Container) (cr []config.ResourceSpec) {
	for _, v := range containers {
		if v.Resources.Requests != nil {
			for resource := range v.Resources.Requests {
				cr = append(cr, config.ResourceSpec{Name: string(resource), Weight: 1})
			}
		}
		if v.Resources.Limits != nil {
			for resource := range v.Resources.Limits {
				if !isExist(string(resource), cr) {
					cr = append(cr, config.ResourceSpec{Name: string(resource), Weight: 1})
				}
			}
		}
	}
	return cr
}
