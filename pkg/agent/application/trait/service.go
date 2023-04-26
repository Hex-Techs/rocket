package trait

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	ServiceKind = "service"
)

func NewServiceTrait() Trait {
	return &ServiceTrait{}
}

var _ Trait = &ServiceTrait{}

type ServiceTrait struct{}

func (*ServiceTrait) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	svc := new(Service)
	if err := yaml.Unmarshal([]byte(ttemp.Template), svc); err != nil {
		if errj := json.Unmarshal([]byte(ttemp.Template), svc); errj != nil {
			return fmt.Errorf("syntax error: %v, %v", err, errj)
		}
	}
	s := &v1.ServiceSpec{
		Ports: []v1.ServicePort{},
	}
	for _, p := range svc.Ports {
		s.Ports = append(s.Ports, v1.ServicePort{
			Name:       p.Name,
			TargetPort: p.TargetPort,
		})
	}
	if svc.Headless {
		s.ClusterIP = "None"
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*s))
	return nil
}

func (st *ServiceTrait) Handler(ttemp *rocketv1alpha1.Trait, workload *rocketv1alpha1.Workload,
	event EventType, client *Client) error {
	name := workload.Name
	namespace := workload.Namespace
	if !workload.DeletionTimestamp.IsZero() || workload == nil {
		if err := client.client.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
	// if workload.Spec.Template.CloneSetTemplate == nil && workload.Spec.Template.StatefulSetTemlate == nil {
	// 	return fmt.Errorf("service trait only support CloneSet and StatefulSet")
	// }
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				constant.ManagedByRocketLabel: "rocket",
			},
		},
	}
	spec := &v1.ServiceSpec{}
	if err := st.Generate(ttemp, spec); err != nil {
		return err
	}
	svc.Spec = *spec
	old, err := client.client.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if event == Created {
			_, err = client.client.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			return err
		}
	}
	if event == Created {
		if !cmp.Equal(old.Spec, svc.Spec) || !cmp.Equal(old.Annotations, svc.Annotations) {
			old.Spec = svc.Spec
			_, err = client.client.CoreV1().Services(namespace).Update(context.TODO(), old, metav1.UpdateOptions{})
			return err
		}
	}
	if event == Deleted {
		err = client.client.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
