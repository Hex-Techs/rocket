package filter

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// 节点容忍度过滤，剔除pod不容忍的节点，返回剩余节点
func tolerateFilter(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) error {
	if nodeInfo == nil || nodeInfo.Node() == nil {
		return fmt.Errorf("invalid nodeInfo")
	}

	filterPredicate := func(t *v1.Taint) bool {
		// PodToleratesNodeTaints is only interested in NoSchedule and NoExecute taints.
		return t.Effect == v1.TaintEffectNoSchedule || t.Effect == v1.TaintEffectNoExecute
	}

	taint, isUntolerated := v1helper.FindMatchingUntoleratedTaint(nodeInfo.Node().Spec.Taints, pod.Spec.Tolerations, filterPredicate)
	if !isUntolerated {
		return nil
	}
	errReason := fmt.Errorf("node(s) had taint {%s: %s}, that the pod didn't tolerate",
		taint.Key, taint.Value)
	return errReason
}
