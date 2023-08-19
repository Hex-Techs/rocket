package filter

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/validation"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// 节点亲和性过滤，选择亲和性的节点并返回
func nodeAffinityFilter(ctx context.Context, pod *v1.Pod, nodeInfo *framework.NodeInfo) error {
	node := nodeInfo.Node()
	if node == nil {
		return errors.New("node not found")
	}
	if pod.Spec.Affinity != nil {
		if pod.Spec.Affinity.NodeAffinity != nil {
			nf := &config.NodeAffinityArgs{AddedAffinity: pod.Spec.Affinity.NodeAffinity}
			args, err := getArgs(nf)
			if err != nil {
				return err
			}
			var addedNodeSelector *nodeaffinity.NodeSelector
			if args.AddedAffinity != nil {
				if ns := args.AddedAffinity.RequiredDuringSchedulingIgnoredDuringExecution; ns != nil {
					addedNodeSelector, err = nodeaffinity.NewNodeSelector(ns)
					if err != nil {
						return fmt.Errorf("parsing addedAffinity.requiredDuringSchedulingIgnoredDuringExecution: %w", err)
					}
				}
			}
			if addedNodeSelector != nil && !addedNodeSelector.Match(node) {
				return errors.New("node(s) didn't match scheduler-enforced node affinity")
			}

			sed := nodeaffinity.GetRequiredNodeAffinity(pod)
			// Ignore parsing errors for backwards compatibility.
			match, _ := sed.Match(node)
			if !match {
				return errors.New("node(s) didn't match Pod's node affinity/selector")
			}
		}
	}
	return nil
}

func getArgs(obj runtime.Object) (config.NodeAffinityArgs, error) {
	ptr, ok := obj.(*config.NodeAffinityArgs)
	if !ok {
		return config.NodeAffinityArgs{}, fmt.Errorf("args are not of type NodeAffinityArgs, got %T", obj)
	}
	return *ptr, validation.ValidateNodeAffinityArgs(nil, ptr)
}
