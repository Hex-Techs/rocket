package filter

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Filter 判断pod是否可以被调度到节点
func Filter(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) bool {
	if err := unScheduler(ctx, pod, nodeInfo); err != nil {
		return false
	}
	if err := tolerateFilter(ctx, pod, nodeInfo); err != nil {
		return false
	}
	if err := nodeAffinityFilter(ctx, pod, nodeInfo); err != nil {
		return false
	}
	return true
}

func Score(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) int64 {
	log := log.FromContext(ctx)
	ret, err := resourceFilter(ctx, pod, nodeInfo)
	if err != nil {
		log.Error(err, "unable to score the node", "nodeName", nodeInfo.Node().Name)
	}
	return ret
}
