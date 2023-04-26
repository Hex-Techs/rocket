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
	ManualScaleKind = "manualscale"
)

func NewMSTrait() Trait {
	return &manualScale{}
}

var _ Trait = &manualScale{}

type manualScale struct{}

func (*manualScale) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	if obj == nil {
		return errors.New("obj is nil")
	}
	ms := new(ManualScale)
	err := yaml.Unmarshal([]byte(ttemp.Template), ms)
	if err != nil {
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
		deploy := &appsv1.Deployment{}
		json.Unmarshal(b, deploy)
		deploy.Spec.Replicas = ms.Replicas
		raw, _ = json.Marshal(deploy)
	case "cloneSets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		clone.Spec.Replicas = ms.Replicas
		if ms.ScaleStrategy != nil {
			clone.Spec.ScaleStrategy.PodsToDelete = ms.ScaleStrategy.PodsToDelete
		}
		raw, _ = json.Marshal(clone)
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			sts.Spec.Replicas = ms.Replicas
			raw, _ = json.Marshal(sts)
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			sts.Spec.Replicas = ms.Replicas
			raw, _ = json.Marshal(sts)
		}
	default:
		return nil, fmt.Errorf("unsupport workload type: %s", gvr.Resource)
	}
	w.Spec.Template.Raw = raw
	return w, nil
}
