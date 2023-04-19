package trait

import (
	"encoding/json"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	TolerateKind = "tolerate"
)

func NewTolerateTrait() Trait {
	return &tolerate{}
}

var _ Trait = &tolerate{}

type tolerate struct{}

func (*tolerate) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	tole := new(Tolerate)
	err := yaml.Unmarshal([]byte(ttemp.Template), tole)
	if err != nil || tole == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), tole)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(tole.Tolerations))
	return nil
}

func (t *tolerate) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, nil
	}
	w := workload
	tole := []v1.Toleration{}
	if err := t.Generate(ttemp, &tole); err != nil {
		return nil, err
	}
	if w.Spec.Template.DeploymentTemplate != nil {
		w.Spec.Template.DeploymentTemplate.Template.Spec.Tolerations = tole
	}
	if w.Spec.Template.CloneSetTemplate != nil {
		w.Spec.Template.CloneSetTemplate.Template.Spec.Tolerations = tole
	}
	if w.Spec.Template.CronJobTemplate != nil {
		w.Spec.Template.CronJobTemplate.JobTemplate.Spec.Template.Spec.Tolerations = tole
	}
	if w.Spec.Template.JobTemplate != nil {
		w.Spec.Template.JobTemplate.Spec.Template.Spec.Tolerations = tole
	}
	if w.Spec.Template.StatefulSetTemlate != nil {
		w.Spec.Template.StatefulSetTemlate.Template.Spec.Tolerations = tole
	}
	if w.Spec.Template.ExtendStatefulSetTemlate != nil {
		w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Tolerations = tole
	}
	return w, nil
}
