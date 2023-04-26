package trait

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	MetricsKind = "metrics"

	EnableAnnotationKey = "prometheus.io/scrape"
	PathAnnotationKey   = "prometheus.io/path"
	PortAnnotationKey   = "prometheus.io/port"
)

func NewMetricsTrait() Trait {
	return &metrics{}
}

var _ Trait = &metrics{}

type metrics struct{}

func (*metrics) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	ms := new(Metrics)
	err := yaml.Unmarshal([]byte(ttemp.Template), ms)
	if err != nil || ms == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), ms)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*ms))
	return nil
}

func (m *metrics) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, nil
	}
	w := workload
	ms := &Metrics{}
	if err := m.Generate(ttemp, ms); err != nil {
		return nil, err
	}
	if ms.Path == "" {
		// 默认 /metrics path
		ms.Path = "/metrics"
	}
	if ms.Port <= 1024 || ms.Port >= 65535 {
		// 端口默认 8090，如果不在范围内就设置为默认端口
		ms.Port = 8090
	}
	resource, gvk, err := gvktools.GetResourceAndGvkFromWorkload(w)
	if err != nil {
		return nil, err
	}
	if resource == nil {
		return nil, errors.New("resource template is nil")
	}
	gvr := gvktools.SetGVRForWorkload(gvk)
	// Convert the resource to JSON bytes
	b, _ := json.Marshal(resource)
	var raw []byte
	switch gvr.Resource {
	case "deployments":
		// Create a new Deployment from the bytes
		deploy := &appsv1.Deployment{}
		json.Unmarshal(b, deploy)
		if len(deploy.Spec.Template.Annotations) == 0 {
			deploy.Spec.Template.Annotations = map[string]string{}
		}
		deploy.Spec.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		deploy.Spec.Template.Annotations[PathAnnotationKey] = ms.Path
		deploy.Spec.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
		// Convert the Deployment back to JSON
		raw, _ = json.Marshal(deploy)
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		if len(clone.Spec.Template.Annotations) == 0 {
			clone.Spec.Template.Annotations = map[string]string{}
		}
		clone.Spec.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		clone.Spec.Template.Annotations[PathAnnotationKey] = ms.Path
		clone.Spec.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
		raw, _ = json.Marshal(clone)
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			if len(sts.Spec.Template.Annotations) == 0 {
				sts.Spec.Template.Annotations = map[string]string{}
			}
			sts.Spec.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
			sts.Spec.Template.Annotations[PathAnnotationKey] = ms.Path
			sts.Spec.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
			raw, _ = json.Marshal(sts)
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			if len(sts.Spec.Template.Annotations) == 0 {
				sts.Spec.Template.Annotations = map[string]string{}
			}
			sts.Spec.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
			sts.Spec.Template.Annotations[PathAnnotationKey] = ms.Path
			sts.Spec.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
			raw, _ = json.Marshal(sts)
		}
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", gvr.Resource)
	}
	// Set the template on the Workload
	w.Spec.Template.Raw = raw
	return w, nil
}
