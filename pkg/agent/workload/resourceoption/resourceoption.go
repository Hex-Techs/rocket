package resourceoption

import rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"

type ResourceOption interface {
	Create(name, namespace string, obj interface{}) error
	Get(name, namespace string, obj interface{}) error
	Delete(name, namespace string) error
	Update(name, namespace string, obj interface{}) error
	Generate(name string, workload *rocketv1alpha1.Workload, obj interface{}) error
}
