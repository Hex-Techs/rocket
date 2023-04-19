package trait

import (
	"encoding/json"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
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
	if w.Spec.Template.DeploymentTemplate != nil {
		if len(w.Spec.Template.DeploymentTemplate.Template.Annotations) == 0 {
			w.Spec.Template.DeploymentTemplate.Template.Annotations = map[string]string{}
		}
		w.Spec.Template.DeploymentTemplate.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		w.Spec.Template.DeploymentTemplate.Template.Annotations[PathAnnotationKey] = ms.Path
		w.Spec.Template.DeploymentTemplate.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
	}
	if w.Spec.Template.CloneSetTemplate != nil {
		if len(w.Spec.Template.CloneSetTemplate.Template.Annotations) == 0 {
			w.Spec.Template.CloneSetTemplate.Template.Annotations = map[string]string{}
		}
		w.Spec.Template.CloneSetTemplate.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		w.Spec.Template.CloneSetTemplate.Template.Annotations[PathAnnotationKey] = ms.Path
		w.Spec.Template.CloneSetTemplate.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
	}
	if w.Spec.Template.StatefulSetTemlate != nil {
		if len(w.Spec.Template.StatefulSetTemlate.Template.Annotations) == 0 {
			w.Spec.Template.StatefulSetTemlate.Template.Annotations = map[string]string{}
		}
		w.Spec.Template.StatefulSetTemlate.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		w.Spec.Template.StatefulSetTemlate.Template.Annotations[PathAnnotationKey] = ms.Path
		w.Spec.Template.StatefulSetTemlate.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
	}
	if w.Spec.Template.ExtendStatefulSetTemlate != nil {
		if len(w.Spec.Template.ExtendStatefulSetTemlate.Template.Annotations) == 0 {
			w.Spec.Template.ExtendStatefulSetTemlate.Template.Annotations = map[string]string{}
		}
		w.Spec.Template.ExtendStatefulSetTemlate.Template.Annotations[EnableAnnotationKey] = fmt.Sprintf("%t", ms.Enable)
		w.Spec.Template.ExtendStatefulSetTemlate.Template.Annotations[PathAnnotationKey] = ms.Path
		w.Spec.Template.ExtendStatefulSetTemlate.Template.Annotations[PortAnnotationKey] = fmt.Sprintf("%d", ms.Port)
	}

	return w, nil
}
