package trait

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/gvktools"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
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
		for i := range deploy.Spec.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				deploy.Spec.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				deploy.Spec.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				deploy.Spec.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
		// Convert the Deployment back to JSON
		raw, _ = json.Marshal(deploy)
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		for i := range clone.Spec.Template.Spec.Containers {
			if probe.LivenessProbe != nil {
				clone.Spec.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
			}
			if probe.ReadinessProbe != nil {
				clone.Spec.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
			}
			if probe.StartupProbe != nil {
				clone.Spec.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
			}
		}
		raw, _ = json.Marshal(clone)
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			for i := range sts.Spec.Template.Spec.Containers {
				if probe.LivenessProbe != nil {
					sts.Spec.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
				}
				if probe.ReadinessProbe != nil {
					sts.Spec.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
				}
				if probe.StartupProbe != nil {
					sts.Spec.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
				}
			}
			raw, _ = json.Marshal(sts)
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			for i := range sts.Spec.Template.Spec.Containers {
				if probe.LivenessProbe != nil {
					sts.Spec.Template.Spec.Containers[i].LivenessProbe = probe.LivenessProbe
				}
				if probe.ReadinessProbe != nil {
					sts.Spec.Template.Spec.Containers[i].ReadinessProbe = probe.ReadinessProbe
				}
				if probe.StartupProbe != nil {
					sts.Spec.Template.Spec.Containers[i].StartupProbe = probe.StartupProbe
				}
			}
			raw, _ = json.Marshal(sts)
		}
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", gvr.Resource)
	}
	// Set the template on the Workload
	w.Spec.Template.Raw = raw
	return w, nil
}
