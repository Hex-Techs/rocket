package trait

import (
	"encoding/json"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	ProbeKind = "probe"
)

func NewProbeTrait() Trait {
	return &probe{}
}

var _ Trait = &probe{}

type probe struct{}

func (*probe) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	probe := new(Probe)
	err := yaml.Unmarshal([]byte(ttemp.Template), probe)
	if err != nil || probe == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), probe)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*probe))
	return nil
}

func (a *probe) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, nil
	}
	w := workload
	probe := &Probe{}
	if err := a.Generate(ttemp, probe); err != nil {
		return nil, err
	}
	if w.Spec.Template.DeploymentTemplate != nil {
		for i := range w.Spec.Template.DeploymentTemplate.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				w.Spec.Template.DeploymentTemplate.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				w.Spec.Template.DeploymentTemplate.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				w.Spec.Template.DeploymentTemplate.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
	}
	if w.Spec.Template.CloneSetTemplate != nil {
		for i := range w.Spec.Template.CloneSetTemplate.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				w.Spec.Template.CloneSetTemplate.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				w.Spec.Template.CloneSetTemplate.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				w.Spec.Template.CloneSetTemplate.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
	}
	if w.Spec.Template.StatefulSetTemlate != nil {
		for i := range w.Spec.Template.StatefulSetTemlate.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				w.Spec.Template.StatefulSetTemlate.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				w.Spec.Template.StatefulSetTemlate.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				w.Spec.Template.StatefulSetTemlate.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
	}
	if w.Spec.Template.ExtendStatefulSetTemlate != nil {
		for i := range w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				w.Spec.Template.ExtendStatefulSetTemlate.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
	}

	return w, nil
}
