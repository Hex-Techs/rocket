package gvktools

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GetResourceAndGvkFromApplication 从 application 中获取资源和 gvk
func GetResourceAndGvkFromApplication(name, namespace string, application *runtime.RawExtension) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	if application.Object == nil && len(application.Raw) == 0 {
		return nil, nil, errors.New("application template is nil")
	}
	// Decode the resource from the application
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(application.Raw, nil, nil)
	if err != nil {
		obj = application.Object
		log.Log.V(3).Info("Failed to decode application template, retry from Object", "error", err)
	}
	res := ConvertToUnstructured(obj)
	res.SetName(name)
	if res == nil {
		return nil, nil, errors.New("application template is nil")
	}
	// NOTE: 为了避免资源的 namespace 与 distribution 的 namespace 不一致，这里需要将资源的 namespace 设置为 distribution 的 namespace
	res.SetNamespace(namespace)
	return res, gvk, nil
}

// SetGVRForDistribution 根据 gvk 设置 gvr
func SetGVRForApplication(gvk *schema.GroupVersionKind) schema.GroupVersionResource {
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
