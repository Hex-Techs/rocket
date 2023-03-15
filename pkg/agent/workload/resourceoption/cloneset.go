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

func NewCloneSetOption(client kclientset.Interface) ResourceOption {
	return &CloneSetOption{
		kclientset: client,
	}
}

var _ ResourceOption = &CloneSetOption{}

type CloneSetOption struct {
	kclientset kclientset.Interface
}

func (c *CloneSetOption) Create(name, namespace string, obj interface{}) error {
	cloneset := &kruiseappsv1alpha1.CloneSet{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(kruiseappsv1alpha1.CloneSet{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(cloneset).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kclientset.AppsV1alpha1().CloneSets(namespace).Create(context.TODO(), cloneset, metav1.CreateOptions{})
	return err
}

func (c *CloneSetOption) Get(name, namespace string, obj interface{}) error {
	cloneset, err := c.kclientset.AppsV1alpha1().CloneSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(cloneset)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*cloneset))
	return nil
}

func (c *CloneSetOption) Delete(name, namespace string) error {
	return c.kclientset.AppsV1alpha1().CloneSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// obj is *cloneset
func (c *CloneSetOption) Update(name, namespace string, obj interface{}) error {
	cloneset := &kruiseappsv1alpha1.CloneSet{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(kruiseappsv1alpha1.CloneSet{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(cloneset).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kclientset.AppsV1alpha1().CloneSets(namespace).Update(context.TODO(), cloneset, metav1.UpdateOptions{})
	return err
}

func (c *CloneSetOption) Generate(name string, workload *rocketv1alpha1.Workload, obj interface{}) error {
	cloneset := &kruiseappsv1alpha1.CloneSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
			},
		},
		Spec: *workload.Spec.Template.CloneSetTemplate,
	}
	if len(cloneset.Labels) == 0 {
		cloneset.Labels = map[string]string{}
	}
	cloneset.Labels[constant.GenerateNameLabel] = name
	cloneset.Labels[constant.WorkloadNameLabel] = workload.Name
	cloneset.Labels[constant.AppNameLabel] = name

	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(cloneset)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*cloneset))
	return nil
}
