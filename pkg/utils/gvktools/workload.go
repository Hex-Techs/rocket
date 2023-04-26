package gvktools

import (
	"errors"
	"fmt"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// GetResourceAndGvkFromWorkload 从 workload 中获取资源和 gvk
func GetResourceAndGvkFromWorkload(workload *rocketv1alpha1.Workload) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	if workload.Spec.Template.Object == nil && len(workload.Spec.Template.Raw) == 0 {
		return nil, nil, errors.New("workload template is nil")
	}
	// Decode the resource from the workload
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(workload.Spec.Template.Raw, nil, nil)
	if err != nil {
		obj = workload.Spec.Template.Object
		klog.V(3).Infof("Failed to decode workload template, err: %v, retry from Object", err)
	}
	res := ConvertToUnstructured(obj)
	if res == nil {
		return nil, nil, errors.New("workload template is nil")
	}
	// NOTE: 为了避免资源的 namespace 与 distribution 的 namespace 不一致，这里需要将资源的 namespace 设置为 distribution 的 namespace
	res.SetNamespace(workload.Namespace)
	return res, gvk, nil
}

// SetGVRForDistribution 根据 gvk 设置 gvr
func SetGVRForWorkload(gvk *schema.GroupVersionKind) schema.GroupVersionResource {
	resourceName := ""
	switch gvk.Kind {
	case "Deployment":
		resourceName = "deployments"
	case "CloneSet":
		resourceName = "clonesets"
	case "StatefulSet":
		resourceName = "statefulsets"
	case "CronJob":
		resourceName = "cronjobs"
	case "Job":
		resourceName = "jobs"
	}
	return gvk.GroupVersion().WithResource(resourceName)
}

// ValidateResource 验证资源是否支持
func ValidateResource(gvr schema.GroupVersionResource) error {
	// supportedResources is a map of supported resources
	supportedResources := map[string]struct{}{
		"deployments":  {},
		"clonesets":    {},
		"statefulsets": {},
		"cronjobs":     {},
		"jobs":         {},
	}
	// Check if the resource is supported
	_, isSupportedResource := supportedResources[gvr.Resource]
	if !isSupportedResource {
		return fmt.Errorf("resource %s is not supported", gvr.Resource)
	}
	// Check if the group is supported for statefulsets
	if gvr.Resource == "statefulsets" && gvr.Group != "apps.kruise.io" {
		return fmt.Errorf("group %s is not supported", gvr.Group)
	}
	return nil
}
