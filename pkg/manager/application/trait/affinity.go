package trait

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	AffinityKind = "affinity"
)

func NewAffiTrait() Trait {
	return &affinity{}
}

var _ Trait = &affinity{}

type affinity struct{}

func (*affinity) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	if obj == nil {
		return errors.New("obj is nil")
	}
	affi := new(Affinity)
	err := yaml.Unmarshal([]byte(ttemp.Template), affi)
	if err != nil || affi == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), affi)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	v1affi := &v1.Affinity{
		NodeAffinity:    affi.NodeAffinity,
		PodAffinity:     affi.PodAffinity,
		PodAntiAffinity: affi.PodAntiAffinity,
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*v1affi))
	return nil
}

func (a *affinity) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, errors.New("workload is nil")
	}
	w := workload
	affi := &v1.Affinity{}
	if err := a.Generate(ttemp, affi); err != nil {
		return nil, err
	}
	if w.Spec.Template.DeploymentTemplate != nil {
		w.Spec.Template.DeploymentTemplate.Template.Spec.Affinity = affi
	}
	if w.Spec.Template.CloneSetTemplate != nil {
		w.Spec.Template.CloneSetTemplate.Template.Spec.Affinity = affi
	}
	if w.Spec.Template.CronJobTemplate != nil {
		w.Spec.Template.CronJobTemplate.JobTemplate.Spec.Template.Spec.Affinity = affi
	}
	if w.Spec.Template.StatefulSetTemlate != nil {
		w.Spec.Template.StatefulSetTemlate.Template.Spec.Affinity = affi
	}
	if w.Spec.Template.ExtendStatefulSetTemlate != nil {
		w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Affinity = affi
	}
	if w.Spec.Template.JobTemplate != nil {
		w.Spec.Template.JobTemplate.Spec.Template.Spec.Affinity = affi
	}
	return w, nil
}
