/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/go-cmp/cmp"
	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/utils/condition"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/syntax"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	TemplateReady  = "TemplateReady"
	ParameterReady = "ParameterReady"
	WorkloadReady  = "WorkloadReady"

	EdgeTraitDelete = "EdgeTraitDelete"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	workloadOption *workloadOption
}

func NewRecociler(mgr manager.Manager) *ApplicationReconciler {
	cli, scheme := mgr.GetClient(), mgr.GetScheme()
	return &ApplicationReconciler{
		Client: cli,
		Scheme: scheme,
		workloadOption: &workloadOption{
			Client: cli,
			Scheme: scheme,
		},
	}
}

//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocket.hextech.io,resources=applications/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	app := &rocketv1alpha1.Application{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, app)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	obj := app.DeepCopy()

	if obj.CreationTimestamp.IsZero() {
		if !tools.ContainsString(obj.Finalizers, constant.ApplicationFinalizer) {
			obj.Finalizers = append(obj.Finalizers, constant.ApplicationFinalizer)
			if err = r.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if tools.ContainsString(obj.Finalizers, constant.ApplicationFinalizer) {
			remove := false
			if obj.Status.TraitCondition != nil {
				if obj.Status.TraitCondition.Reason == EdgeTraitDelete {
					remove = true
				}
			} else {
				remove = true
			}
			if remove {
				obj.Finalizers = tools.RemoveString(obj.Finalizers, constant.ApplicationFinalizer)
				if err = r.Update(ctx, obj); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
	}

	obj.Status.Conditions = []metav1.Condition{}

	// validate
	kind, conditions := r.validateTemplateAndParameter(obj)
	obj.Status.Conditions = append(obj.Status.Conditions, conditions...)

	isHandleWorkload := true
	for _, condition := range obj.Status.Conditions {
		if condition.Status == metav1.ConditionFalse {
			isHandleWorkload = false
			break
		}
	}
	if isHandleWorkload {
		obj.Status.Conditions = append(obj.Status.Conditions, r.hendleWorkload(kind, app))
	}

	// update status
	err = r.Status().Update(ctx, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketv1alpha1.Application{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				old := e.ObjectOld.(*rocketv1alpha1.Application)
				new := e.ObjectNew.(*rocketv1alpha1.Application)
				if !cmp.Equal(old.Annotations, new.Annotations) {
					return true
				}
				if !cmp.Equal(old.Labels, new.Labels) {
					return true
				}
				if !cmp.Equal(old.Spec, new.Spec) {
					return true
				}
				tmpOldStatus, tmpNewStatus := old.Status.Conditions, new.Status.Conditions
				old.Status.Conditions = []metav1.Condition{}
				new.Status.Conditions = []metav1.Condition{}
				resldualOld, resldualNew := old.Status, new.Status
				if !cmp.Equal(resldualOld, resldualNew) {
					return true
				}
				return condition.ConditionNotEqual(tmpOldStatus, tmpNewStatus)
			},
		})).
		Owns(&rocketv1alpha1.Workload{}).
		Watches(source.NewKindWithCache(&rocketv1alpha1.Template{}, mgr.GetCache()), handler.Funcs{
			UpdateFunc: func(ue event.UpdateEvent, q workqueue.RateLimitingInterface) {
				// template更新后触发application的更新操作，只有在强制刷新时可用
				old := ue.ObjectOld.(*rocketv1alpha1.Template)
				for _, v := range old.Status.UsedBy {
					ac := &rocketv1alpha1.Application{}
					err := r.Get(context.Background(), types.NamespacedName{Namespace: old.Namespace, Name: v.Name}, ac)
					if err != nil {
						klog.V(0).Infof("Template '%s/%s' was changed, but get Application '%s/%s' with error: %v",
							old.Namespace, old.Name, old.Namespace, v.Name, err)
						continue
					}
					// NOTE: 当flush这个数据存在，无论值如何都会触发更新。不存在时则不触发更新
					if _, ok := ac.Annotations[constant.FlushAnnotation]; ok {
						q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: old.Namespace, Name: v.Name}})
					}
				}
			},
		}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size for default ratelimiter workqueue
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)}).
		Complete(r)
}

func (r *ApplicationReconciler) getTemplates(ac *rocketv1alpha1.Application) (
	templates []*rocketv1alpha1.Template, images map[string][]rocketv1alpha1.ImageDefine, err error) {
	images = map[string][]rocketv1alpha1.ImageDefine{}
	for _, c := range ac.Spec.Templates {
		temp := &rocketv1alpha1.Template{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: ac.Namespace, Name: c.Name}, temp)
		if err != nil {
			return templates, images, err
		}
		images[c.Name] = c.ImageDefine
		templates = append(templates, temp)
	}
	return templates, images, nil
}

func (r *ApplicationReconciler) validateTemplateAndParameter(app *rocketv1alpha1.Application) (
	rocketv1alpha1.WorkloadType, []metav1.Condition) {
	conditions := make([]metav1.Condition, 2)
	conditions[0] = condition.GenerateCondition(TemplateReady, "TemplateAvailable", "all templates available", metav1.ConditionTrue)
	conditions[1] = condition.GenerateCondition(ParameterReady, "ParameterAvailable", "all parameters available", metav1.ConditionTrue)
	intersectionArea := mapset.NewSet[string]()
	kind := rocketv1alpha1.WorkloadType("")
	templates, images, err := r.getTemplates(app)
	if err != nil {
		return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable", err.Error(), metav1.ConditionFalse)}
	}
	for idx, temp := range templates {
		if idx > 0 {
			// 所有 template 的 workloadType 必须相同
			if kind != temp.Spec.WorkloadType {
				return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable",
					"workloadtypes of all templates must be the same", metav1.ConditionFalse)}
			}
		}
		kind = temp.Spec.WorkloadType
		app.Status.WorkloadType = kind

		if err := r.validateTemplateOverlap(temp); err != nil {
			return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable", err.Error(), metav1.ConditionFalse)}
		}

		if err := r.validateTemplateImage(temp, images[temp.Name]); err != nil {
			return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable", err.Error(), metav1.ConditionFalse)}
		}

		areaSet := mapset.NewSet[string]()
		for _, area := range temp.Spec.ApplyScope.CloudAreas {
			areaSet.Add(area)
		}
		if idx == 0 {
			intersectionArea = areaSet
		}
		intersectionArea = intersectionArea.Intersect(areaSet)
		pSet := mapset.NewSet[string]()
		for _, val := range temp.Spec.Parameters {
			pSet.Add(val.Name)
		}
		for _, val := range app.Spec.Templates[idx].ParameterValues {
			if syntax.SyntaxEngine.ValidateSyntax(val.Value) {
				if !pSet.Contains(val.Name) {
					conditions[1] = condition.GenerateCondition(TemplateReady, "ParameterAvailable",
						fmt.Sprintf("template '%s' parameter cat not found '%s' in parameters", temp.Name, val.Name),
						metav1.ConditionFalse)
					return kind, conditions
				}
			}
		}
		err = r.validateParameterRequired(app.Spec.Variables, app.Spec.Templates[idx].ParameterValues, temp)
		if err != nil {
			conditions[1] = condition.GenerateCondition(TemplateReady, "ParameterAvailable",
				fmt.Sprintf("template '%s' %v", temp.Name, err),
				metav1.ConditionFalse)
			return kind, conditions
		}
	}
	// 验证是否在同一 area 中工作
	if intersectionArea.Cardinality() == 0 {
		return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable",
			"all templates cannot work in the same cloud area", metav1.ConditionFalse)}
	}
	if !intersectionArea.Contains(app.Spec.CloudArea) {
		return kind, []metav1.Condition{condition.GenerateCondition(TemplateReady, "TemplateAvailable",
			fmt.Sprintf("all templates cannot work in '%s' cloud area", app.Spec.CloudArea), metav1.ConditionFalse)}
	}
	return kind, conditions
}

// validate the parameter, and validate parameter type
func (r *ApplicationReconciler) validateParameterRequired(variables []rocketv1alpha1.Variable,
	parameters []rocketv1alpha1.ParameterValue, temp *rocketv1alpha1.Template) error {
	pset := map[string]string{}
	for _, acp := range parameters {
		pset[acp.Name] = acp.Value
	}
	vset := map[string]string{}
	for _, v := range variables {
		vset[v.Name] = v.Value
	}
	for _, param := range temp.Spec.Parameters {
		if v, ok := pset[param.Name]; !ok {
			// 必要参数如果没有默认值且在配置中没有设置，则会报错
			if param.Required && param.Default == "" {
				return fmt.Errorf("'%s' parameter was required but not set default", param.Name)
			}
		} else {
			name := syntax.SyntaxEngine.GetVar(v)
			val := vset[name]
			if param.Type == "number" {
				_, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return fmt.Errorf(
						"'%s' does not match type, type is number, value '%s' can not parse to number", param.Name, val)
				}
			}
			if param.Type == "boolean" {
				if val != "true" && val != "false" {
					return fmt.Errorf("'%s' does not match type, type boolean must be 'true' or 'false', got %s", param.Name, val)
				}
			}
		}
	}
	return nil
}

func (r *ApplicationReconciler) validateTemplateOverlap(temp *rocketv1alpha1.Template) error {
	if !temp.Spec.ApplyScope.AllowOverlap &&
		temp.Status.UsedAgain == metav1.ConditionFalse &&
		temp.Status.NumberOfWorkload > 1 {
		return fmt.Errorf("reuse of template '%s' is not allowed", temp.Name)
	}
	return nil
}

func (r *ApplicationReconciler) validateTemplateImage(temp *rocketv1alpha1.Template, images []rocketv1alpha1.ImageDefine) error {
	set := mapset.NewSet[string]()
	for _, c := range temp.Spec.Containers {
		set.Add(c.Name)
	}
	for _, i := range images {
		if !set.Contains(i.ContainerName) {
			return fmt.Errorf("not found container '%s' by image", i.ContainerName)
		}
	}
	return nil
}

func (r *ApplicationReconciler) hendleWorkload(kind rocketv1alpha1.WorkloadType, app *rocketv1alpha1.Application) metav1.Condition {
	// 1. get workload list
	ws := &rocketv1alpha1.WorkloadList{}
	err := r.List(context.TODO(), ws, &client.ListOptions{Namespace: app.Namespace})
	if err != nil {
		return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", err.Error(), metav1.ConditionFalse)
	}
	// 2. generate workload
	workload, err := r.workloadOption.generateWorkload(kind, app)
	if err != nil {
		return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", err.Error(), metav1.ConditionFalse)
	}
	// 3. get workload for current app
	workloadOwns := []rocketv1alpha1.Workload{}
	for _, w := range ws.Items {
		if metav1.IsControlledBy(&w, app) {
			workloadOwns = append(workloadOwns, w)
		}
	}
	// 4. 删除多余的workload
	var old *rocketv1alpha1.Workload
	if len(workloadOwns) > 0 {
		// 处理多余的 workload，可能是用户手动创建的
		if len(workloadOwns) > 1 {
			for _, w := range workloadOwns[1:] {
				err = r.Delete(context.TODO(), &w)
				if err != nil {
					return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", err.Error(), metav1.ConditionFalse)
				}
			}
		}
		old = &workloadOwns[0]
	}
	if old != nil {
		// Cluster 信息保持原状
		workload.Status.Clusters = old.Status.Clusters
		if !cmp.Equal(old.Spec, workload.Spec) ||
			!cmp.Equal(old.Labels, workload.Labels) ||
			!cmp.Equal(old.Annotations, workload.Annotations) {
			old.Spec = workload.Spec
			old.Labels = workload.Labels
			old.Annotations = workload.Annotations
			err = r.Update(context.TODO(), old)
			if err != nil {
				msg := fmt.Sprintf("update workload failed, %v", err)
				return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", msg, metav1.ConditionFalse)
			}
			return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", "workload synced", metav1.ConditionTrue)
		} else {
			klog.V(4).Infof("workload '%s/%s' is not changed", workload.Namespace, workload.Name)
			return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", "workload synced", metav1.ConditionTrue)
		}
	}
	err = r.Create(context.TODO(), workload)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", err.Error(), metav1.ConditionFalse)
		}
	}
	return condition.GenerateCondition(WorkloadReady, "WorkloadSyncedReady", "workload synced", metav1.ConditionTrue)
}
