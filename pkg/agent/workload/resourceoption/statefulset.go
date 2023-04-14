package resourceoption

import (
	"context"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewStatefulSetOption(client kclientset.Interface) ResourceOption {
	return &StatefulSetOption{
		kclientset: client,
	}
}

var _ ResourceOption = &StatefulSetOption{}

type StatefulSetOption struct {
	kclientset kclientset.Interface
}

func (c *StatefulSetOption) Create(name, namespace string, obj interface{}) error {
	statefulset := &kruiseappsv1alpha1.StatefulSet{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(kruiseappsv1alpha1.StatefulSet{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(statefulset).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kclientset.AppsV1alpha1().StatefulSets(namespace).Create(context.TODO(), statefulset, metav1.CreateOptions{})
	return err
}

func (c *StatefulSetOption) Get(name, namespace string, obj interface{}) error {
	statefulset, err := c.kclientset.AppsV1alpha1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(statefulset)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*statefulset))
	return nil
}

func (c *StatefulSetOption) Delete(name, namespace string) error {
	return c.kclientset.AppsV1alpha1().StatefulSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// obj is *statefulset
func (c *StatefulSetOption) Update(name, namespace string, obj interface{}) error {
	statefulset := &kruiseappsv1alpha1.StatefulSet{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(kruiseappsv1alpha1.StatefulSet{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(statefulset).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kclientset.AppsV1alpha1().StatefulSets(namespace).Update(context.TODO(), statefulset, metav1.UpdateOptions{})
	return err
}

func (c *StatefulSetOption) Generate(name string, workload *rocketv1alpha1.Workload, obj interface{}) error {
	statefulset := &kruiseappsv1alpha1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
			},
		},
		Spec: *workload.Spec.Template.ExtendStatefulSetTemlate,
	}
	if len(statefulset.Labels) == 0 {
		statefulset.Labels = map[string]string{}
	}
	statefulset.Labels[constant.GenerateNameLabel] = name
	statefulset.Labels[constant.WorkloadNameLabel] = workload.Name
	statefulset.Labels[constant.AppNameLabel] = name

	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(statefulset)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*statefulset))
	return nil
}
