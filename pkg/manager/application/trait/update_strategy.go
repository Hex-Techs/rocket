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
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	UpdateStrategyKind               = "updatestrategy"
	CloneSetUpdateStrategyVersion    = "apps.kruise.io/cloneset"
	StatefulSetUpdateStrategyVersion = "apps.kruise.io/statefulset"
)

func NewUSTrait() Trait {
	return &updateStrategy{}
}

var _ Trait = &updateStrategy{}

type updateStrategy struct{}

func (u *updateStrategy) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	switch ttemp.Version {
	case CloneSetUpdateStrategyVersion:
		csus, err := u.generateClonesetUpdateStrategy(ttemp)
		if err != nil {
			return err
		}
		mid := reflect.ValueOf(obj).Elem()
		mid.Set(reflect.ValueOf(*csus))
	case StatefulSetUpdateStrategyVersion:
		ssus, err := u.generateStatefulsetUpdateStrategy(ttemp)
		if err != nil {
			return err
		}
		mid := reflect.ValueOf(obj).Elem()
		mid.Set(reflect.ValueOf(*ssus))
	}
	return nil
}

func (u *updateStrategy) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, nil
	}
	w := workload
	resource, gvk, err := gvktools.GetResourceAndGvkFromWorkload(w)
	if err != nil {
		return nil, err
	}
	if resource == nil {
		return nil, errors.New("resource template is nil")
	}
	gvr := gvktools.SetGVRForWorkload(gvk)
	if err := gvktools.ValidateResource(gvr); err != nil {
		return nil, err
	}
	// Convert the resource to JSON bytes
	b, _ := json.Marshal(resource)
	var raw []byte
	if ttemp.Version == CloneSetUpdateStrategyVersion {
		csus := &kruiseappsv1alpha1.CloneSetUpdateStrategy{}
		if err := u.Generate(ttemp, csus); err != nil {
			return nil, err
		}
		if gvr.Resource == "clonesets" {
			clone := &kruiseappsv1alpha1.CloneSet{}
			json.Unmarshal(b, clone)
			clone.Spec.UpdateStrategy = *csus
			raw, _ = json.Marshal(clone)
		}
	}
	if ttemp.Version == StatefulSetUpdateStrategyVersion {
		ssus := &kruiseappsv1beta1.StatefulSetUpdateStrategy{}
		if err := u.Generate(ttemp, ssus); err != nil {
			return nil, err
		}
		if gvr.Resource == "statefulsets" && gvr.Group == "apps.kruise.io" {
			sts := &kruiseappsv1beta1.StatefulSet{}
			json.Unmarshal(b, sts)
			sts.Spec.UpdateStrategy = *ssus
			raw, _ = json.Marshal(sts)
		}
	}
	// Set the template on the Workload
	w.Spec.Template.Raw = raw
	return w, nil
}

// 兼容 yaml 和 json 格式的配置
func (u *updateStrategy) generateClonesetUpdateStrategy(ttemp *rocketv1alpha1.Trait) (
	*kruiseappsv1alpha1.CloneSetUpdateStrategy, error) {
	csus := new(kruiseappsv1alpha1.CloneSetUpdateStrategy)
	err := yaml.Unmarshal([]byte(ttemp.Template), csus)
	if err != nil || csus == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), csus)
		if errj != nil {
			return nil, fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	return csus, nil
}

func (u *updateStrategy) generateStatefulsetUpdateStrategy(ttemp *rocketv1alpha1.Trait) (
	*kruiseappsv1alpha1.StatefulSetUpdateStrategy, error) {
	ssus := new(kruiseappsv1alpha1.StatefulSetUpdateStrategy)
	err := yaml.Unmarshal([]byte(ttemp.Template), ssus)
	if err != nil || ssus == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), ssus)
		if errj != nil {
			return nil, fmt.Errorf("synax error: %v, %v", err, errj)
		}
		return nil, err
	}
	return ssus, nil
}
