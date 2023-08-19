package trait

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/gvktools"
	kruisepolicyv1alpha1 "github.com/openkruise/kruise-api/policy/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	PodUnavailableBudgetKind = "podunavailablebudget"
)

func NewPubTrait() Trait {
	return &pub{}
}

var _ Trait = &pub{}

// 只针对于 cloneset 有效
type pub struct{}

func (*pub) Generate(ttemp *rocketv1alpha1.Trait, obj interface{}) error {
	pub := new(PodUnavailableBudget)
	err := yaml.Unmarshal([]byte(ttemp.Template), pub)
	if err != nil || pub == nil {
		errj := json.Unmarshal([]byte(ttemp.Template), pub)
		if errj != nil {
			return fmt.Errorf("synax error: %v, %v", err, errj)
		}
	}
	policyPub := &kruisepolicyv1alpha1.PodUnavailableBudgetSpec{
		MaxUnavailable: pub.MaxUnavailable,
		MinAvailable:   pub.MinAvailable,
	}
	mid := reflect.ValueOf(obj).Elem()
	mid.Set(reflect.ValueOf(*policyPub))
	return nil
}

func (p *pub) Handler(ttemp *rocketv1alpha1.Trait, app *rocketv1alpha1.Application,
	event EventType, client *Client) error {
	namespace := app.Namespace
	name := app.Name
	// workload 已删除，不需要判断，直接删除
	if !app.DeletionTimestamp.IsZero() || app == nil {
		err := client.Kclient.PolicyV1alpha1().PodUnavailableBudgets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
	pub := &kruisepolicyv1alpha1.PodUnavailableBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				constant.ManagedByRocketLabel: "rocket",
			},
		},
	}
	spec := &kruisepolicyv1alpha1.PodUnavailableBudgetSpec{}
	if err := p.Generate(ttemp, spec); err != nil {
		return err
	}
	pub.Spec = *spec
	pub.Spec.TargetReference = generateTarget(app)
	if pub.Spec.TargetReference == nil {
		return fmt.Errorf("generate target reference failed")
	}
	old, err := client.Kclient.PolicyV1alpha1().PodUnavailableBudgets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if event == CreatedOrUpdate {
			_, err = client.Kclient.PolicyV1alpha1().PodUnavailableBudgets(namespace).Create(context.TODO(), pub, metav1.CreateOptions{})
			return err
		}
	}
	if event == CreatedOrUpdate {
		if !cmp.Equal(old.Spec, pub.Spec) || !cmp.Equal(old.Annotations, pub.Annotations) {
			old.Spec = pub.Spec
			_, err = client.Kclient.PolicyV1alpha1().PodUnavailableBudgets(namespace).Update(context.TODO(), old, metav1.UpdateOptions{})
			return err
		}
	}
	if event == Deleted {
		err := client.Kclient.PolicyV1alpha1().PodUnavailableBudgets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func generateTarget(app *rocketv1alpha1.Application) *kruisepolicyv1alpha1.TargetReference {
	target := &kruisepolicyv1alpha1.TargetReference{
		Name: app.Name,
	}
	_, gvk, err := gvktools.GetResourceAndGvkFromApplication(app)
	if err != nil {
		log.Log.Error(err, "get resource and gvk from workload failed")
		return nil
	}
	gvr := gvktools.SetGVRForApplication(gvk)
	switch gvr.Resource {
	case "deployments":
		target.APIVersion = "apps/v1"
		target.Kind = "Deployment"
	case "clonesets":
		target.APIVersion = "apps.kruise.io/v1alpha1"
		target.Kind = "CloneSet"
	case "statefulsets":
		if gvr.Group == "apps" {
			target.APIVersion = "apps/v1"
			target.Kind = "StatefulSet"
		} else {
			target.APIVersion = "apps.kruise.io/v1beta1"
			target.Kind = "StatefulSet"
		}
	}
	return target
}
