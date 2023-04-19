package resourceoption

import (
	"context"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewDeploymentOption(client kubernetes.Interface) ResourceOption {
	return &DeploymentOption{
		client: client,
	}
}

var _ ResourceOption = &DeploymentOption{}

type DeploymentOption struct {
	client kubernetes.Interface
}

func (c *DeploymentOption) Create(name, namespace string, obj interface{}) error {
	Deployment := &appsv1.Deployment{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(appsv1.Deployment{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(Deployment).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.client.AppsV1().Deployments(namespace).Create(context.TODO(), Deployment, metav1.CreateOptions{})
	return err
}

func (c *DeploymentOption) Get(name, namespace string, obj interface{}) error {
	Deployment, err := c.client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(Deployment)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*Deployment))
	return nil
}

func (c *DeploymentOption) Delete(name, namespace string) error {
	return c.client.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// obj is *Deployment
func (c *DeploymentOption) Update(name, namespace string, obj interface{}) error {
	Deployment := &appsv1.Deployment{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(appsv1.Deployment{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(Deployment).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.client.AppsV1().Deployments(namespace).Update(context.TODO(), Deployment, metav1.UpdateOptions{})
	return err
}

func (c *DeploymentOption) Generate(name string, workload *rocketv1alpha1.Workload, obj interface{}) error {
	Deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
			},
		},
		Spec: *workload.Spec.Template.DeploymentTemplate,
	}
	if len(Deployment.Labels) == 0 {
		Deployment.Labels = map[string]string{}
	}
	Deployment.Labels[constant.GenerateNameLabel] = name
	Deployment.Labels[constant.WorkloadNameLabel] = workload.Name
	Deployment.Labels[constant.AppNameLabel] = name

	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(Deployment)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*Deployment))
	return nil
}
