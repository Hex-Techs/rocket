package trait

import (
	"encoding/json"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	ManualScaleKind = "manualscale"
)

func NewMSTrait() Trait {
	return &manualScale{}
}

var _ Trait = &manualScale{}

type manualScale struct{}

func (*manualScale) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	ms := new(ManualScale)
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

func (m *manualScale) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, nil
	}
	w := workload
	ms := &ManualScale{}
	if err := m.Generate(ttemp, ms); err != nil {
		return nil, err
	}
	if w.Spec.Template.DeploymentTemplate != nil {
		w.Spec.Template.DeploymentTemplate.Replicas = ms.Replicas
	}
	if w.Spec.Template.CloneSetTemplate != nil {
		w.Spec.Template.CloneSetTemplate.Replicas = ms.Replicas
		if ms.ScaleStrategy != nil {
			w.Spec.Template.CloneSetTemplate.ScaleStrategy.PodsToDelete = ms.ScaleStrategy.PodsToDelete
		}
	}
	if w.Spec.Template.StatefulSetTemlate != nil {
		w.Spec.Template.StatefulSetTemlate.Replicas = ms.Replicas
	}
	if w.Spec.Template.ExtendStatefulSetTemlate != nil {
		w.Spec.Template.ExtendStatefulSetTemlate.Replicas = ms.Replicas
	}
	return w, nil
}
