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
	batchv1 "k8s.io/api/batch/v1"
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
		// Set the affinity on the Deployment
		deploy.Spec.Template.Spec.Affinity = affi
		// Convert the Deployment back to JSON
		raw, _ = json.Marshal(deploy)
	case "clonesets":
		clone := &kruiseappsv1alpha1.CloneSet{}
		json.Unmarshal(b, clone)
		clone.Spec.Template.Spec.Affinity = affi
		raw, _ = json.Marshal(clone)
	case "statefulsets":
		if gvr.Group == "apps" {
			sts := &appsv1.StatefulSet{}
			json.Unmarshal(b, sts)
			sts.Spec.Template.Spec.Affinity = affi
			raw, _ = json.Marshal(sts)
		}
		if gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			sts.Spec.Template.Spec.Affinity = affi
			raw, _ = json.Marshal(sts)
		}
	case "cronjobs":
		cj := &batchv1.CronJob{}
		json.Unmarshal(b, cj)
		cj.Spec.JobTemplate.Spec.Template.Spec.Affinity = affi
		raw, _ = json.Marshal(cj)
	case "jobs":
		j := &batchv1.Job{}
		json.Unmarshal(b, j)
		j.Spec.Template.Spec.Affinity = affi
		raw, _ = json.Marshal(j)
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", gvr.Resource)
	}
	// Set the template on the Workload
	w.Spec.Template.Raw = raw
	return w, nil
}
