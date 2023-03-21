package schedule

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// 不满足调度条件的节点过滤掉
func unScheduler(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) error {
	if nodeInfo == nil || nodeInfo.Node() == nil {
		return errors.New("node(s) had unknown conditions")
	}
	// If pod tolerate unschedulable taint, it's also tolerate `node.Spec.Unschedulable`.
	podToleratesUnschedulable := v1helper.TolerationsTolerateTaint(pod.Spec.Tolerations, &v1.Taint{
		Key:    v1.TaintNodeUnschedulable,
		Effect: v1.TaintEffectNoSchedule,
	})
	if nodeInfo.Node().Spec.Unschedulable && !podToleratesUnschedulable {
		return errors.New("node(s) were unschedulable")
	}
	return nil
}
