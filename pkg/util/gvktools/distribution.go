package gvktools

import (
	"fmt"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/config"
	"github.com/hex-techs/rocket/pkg/util/constant"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// GetResourceAndGvkFromDistribution 从 distribution 中获取资源和 gvk
func GetResourceAndGvkFromDistribution(rd *rocketv1alpha1.Distribution) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	// Decode the resource from the distribution
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(rd.Spec.Resource.Raw, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	res := ConvertToUnstructured(obj)
	// NOTE: 为了避免资源的 namespace 与 distribution 的 namespace 不一致，这里需要将资源的 namespace 设置为 distribution 的 namespace
	res.SetNamespace(rd.Namespace)
	// NOTE: 如果是 namespace 类型的资源，需要添加 delete-protection 的 label
	if res.GetKind() == "Namespace" {
		res.SetLabels(map[string]string{
			constant.ManagedByRocketLabel:        "rocket",
			"policy.kruise.io/delete-protection": "Always",
		})
	} else {
		res.SetLabels(map[string]string{
			constant.ManagedByRocketLabel: "rocket",
		})
	}
	return res, gvk, nil
}

// bool: is deploy in current cluster
func Match(rd *rocketv1alpha1.Distribution) (*unstructured.Unstructured, *schema.GroupVersionKind, bool, error) {
	// 获取 distribution 的集群信息
	var m bool
	if rd.Spec.Targets.All != nil {
		if *rd.Spec.Targets.All {
			m = true
		}
	}
	if len(rd.Spec.Targets.IncludedClusters.List) != 0 {
		for i, v := range rd.Spec.Targets.IncludedClusters.List {
			if v.Name == config.Pread().Name {
				m = true
				break
			}
			if i == len(rd.Spec.Targets.IncludedClusters.List)-1 {
				m = false
			}
		}
	}
	resource, gvk, err := GetResourceAndGvkFromDistribution(rd)
	return resource, gvk, m, err
}

func SetGVRForDistribution(gvk *schema.GroupVersionKind) schema.GroupVersionResource {
	resourceName := ""
	switch gvk.Kind {
	case "Namespace":
		resourceName = "namespaces"
	case "ConfigMap":
		resourceName = "configmaps"
	case "Secret":
		resourceName = "secrets"
	case "ResourceQuota":
		resourceName = "resourcequotas"
	}
	return gvk.GroupVersion().WithResource(resourceName)
}

// NamespacedScope 判断资源是否是 namespace 类型的资源
func NamespacedScope(rd *unstructured.Unstructured, cli *discovery.DiscoveryClient) (bool, error) {
	gvk := rd.GetObjectKind().GroupVersionKind()
	list, err := cli.ServerPreferredResources()
	if err != nil {
		return false, err
	}
	for _, v := range list {
		if len(v.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(v.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range v.APIResources {
			if gv.String() == gvk.GroupVersion().String() && r.Kind == gvk.Kind {
				return r.Namespaced, nil
			}
		}
	}
	return false, fmt.Errorf("unknow resource %s", gvk.Kind)
}
