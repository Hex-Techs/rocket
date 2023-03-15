package resourceoption

import (
	"context"
	"fmt"
	"reflect"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/util/constant"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewCronJobOption(client kubernetes.Interface) ResourceOption {
	return &CronJobOption{
		kubeclientset: client,
	}
}

var _ ResourceOption = &CronJobOption{}

// NOTE: k8s 坑人，cronjob 是 v1beta1，job 确是 v1，不知道什么时候会改。current: version 1.20
type CronJobOption struct {
	kubeclientset kubernetes.Interface
}

func (c *CronJobOption) Create(name, namespace string, obj interface{}) error {
	cj := &batchv1.CronJob{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(batchv1.CronJob{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(cj).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kubeclientset.BatchV1().CronJobs(namespace).Create(context.TODO(), cj, metav1.CreateOptions{})
	return err
}

func (c *CronJobOption) Get(name, namespace string, obj interface{}) error {
	cj, err := c.kubeclientset.BatchV1().CronJobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(cj)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*cj))
	return nil
}

func (c *CronJobOption) Delete(name, namespace string) error {
	return c.kubeclientset.BatchV1().CronJobs(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// obj is *cronjob
func (c *CronJobOption) Update(name, namespace string, obj interface{}) error {
	cj := &batchv1.CronJob{}
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(batchv1.CronJob{})
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(cj).Elem()
	mid.Set(reflect.ValueOf(obj))
	_, err := c.kubeclientset.BatchV1().CronJobs(namespace).Update(context.TODO(), cj, metav1.UpdateOptions{})
	return err
}

func (c *CronJobOption) Generate(name string, workload *rocketv1alpha1.Workload, obj interface{}) error {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.Namespace,
			Labels:    workload.Labels,
			Annotations: map[string]string{
				constant.WorkloadNameLabel: workload.Name,
			},
		},
		Spec: *workload.Spec.Template.CronJobTemplate,
	}
	if len(cj.Labels) == 0 {
		cj.Labels = map[string]string{}
	}
	cj.Labels[constant.GenerateNameLabel] = name
	cj.Labels[constant.WorkloadNameLabel] = workload.Name
	vType := reflect.TypeOf(obj)
	valueType := reflect.TypeOf(cj)
	if vType != valueType {
		return fmt.Errorf("%v is not type %v", valueType, vType)
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*cj))
	return nil
}
