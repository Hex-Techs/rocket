package filter

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	schedutil "k8s.io/kubernetes/pkg/scheduler/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// resourceToWeightMap 包含资源名称和权重.
type resourceToWeightMap map[v1.ResourceName]int64

// resourceToValueMap 资源名称和数量.
type resourceToValueMap map[v1.ResourceName]int64

// 节点资源过滤，计算节点可以分配的pod数量
func resourceFilter(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) (int64, error) {
	log := log.FromContext(ctx)
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
	log.V(4).Info("node resource information", "nodeName", node.Name, "Request", request, "Requested", requested, "Allocatable", allocatable)
	replicas := calculateNodeCanAllocatePodReplicas(request, requested, allocatable)
	log.V(4).Info("number of node that can be allocated", "nodeName", node.Name, "Replicas", replicas)
	return replicas, nil
}

// 计算节点可以分配的pod数量
func calculateNodeCanAllocatePodReplicas(request, requested, allocable resourceToValueMap) int64 {
	log := log.FromContext(context.Background())
	var minReplicas []int64
	for resource := range request {
		requestValue := request[resource]
		requestedValue := requested[resource]
		allocableValue := allocable[resource]
		if allocableValue < requestValue {
			log.V(4).Info("the available CPU resources are insufficient", "Request", requestValue, "Allocable", allocableValue, "resourceName", resource)
			return 0
		}
		if allocableValue < requestedValue {
			log.V(4).Info("calculate error requested value are insufficient", "Requestd", requestedValue, "Allocable", allocableValue, "resourceName", resource)
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

// calculatePodResourceRequest 返回非0请求总数. 目前liuer平台没有使用pod overhead（开销）特性
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

	// 关于pod开销的计算，liuer平台没有使用pod overhead（开销）特性
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
	log := log.FromContext(context.Background())
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
	log.V(5).Info("Requested resource is omitted for node score calculation", "resourceName", resource)
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
