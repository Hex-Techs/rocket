package trait

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	DeletionProtectionKind = "deletionprotection"

	Retry = "retry"

	DeleteProtectionLabel = "policy.kruise.io/delete-protection"
)

func NewDPTrait() Trait {
	return &deletionProtection{}
}

var _ Trait = &deletionProtection{}

// 目前只支持 cloneset、statefulset 资源，不支持 cronjob
type deletionProtection struct{}

func (*deletionProtection) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	if obj == nil {
		return errors.New("obj is nil")
	}
	dp := new(DeletionProtection)
	err := yaml.Unmarshal([]byte(ttemp.Template), dp)
	if err != nil || dp == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), dp)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	if dp.Type != "Always" && dp.Type != "Cascading" {
		return fmt.Errorf("delete protection type error: must be 'Always' or 'Cascading', but got '%s'", dp.Type)
	}
	m := obj.(*map[string]string)
	newlabels := map[string]string{
		DeleteProtectionLabel: dp.Type,
	}
	for k, v := range *m {
		newlabels[k] = v
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(newlabels))
	return nil
}

func (d *deletionProtection) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload) (*rocketv1alpha1.Workload, error) {
	if workload == nil {
		return nil, errors.New("workload is nil")
	}
	m := workload.Labels
	if err := d.Generate(ttemp, &m); err != nil {
		return nil, err
	}
	workload.Labels = m
	return workload, nil
}
