package application

import (
	"context"
	"encoding/json"
	"fmt"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/manager/application/trait"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	APIVersion = "rocket.hextech.io/v1alpha1"
	Kind       = "Application"
)

type workloadOption struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *workloadOption) generateWorkload(kind rocketv1alpha1.WorkloadType, app *rocketv1alpha1.Application) (
	*rocketv1alpha1.Workload, error) {
	labels := map[string]string{}
	if app.Labels != nil {
		labels = app.Labels
	}
	labels[constant.AppNameLabel] = app.Name
	// workload继承app的annotation和label，workload名称直接使用app的名称
	workload := &rocketv1alpha1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   app.Namespace,
			Name:        app.Name,
			Annotations: app.Annotations,
			Labels:      labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: APIVersion,
					Kind:       Kind,
					Name:       app.Name,
					UID:        app.UID,
					Controller: pointer.Bool(true),
				},
			},
		},
		Spec: rocketv1alpha1.WorkloadSpec{
			Regions:     app.Spec.Regions,
			Template:    runtime.RawExtension{},
			Tolerations: app.Spec.Tolerations,
		},
	}
	var raw []byte
	switch kind {
	case rocketv1alpha1.Stateless:
		if _, ok := app.Annotations[constant.ExtendedResourceAnnotation]; ok {
			cloneset, err := r.generateCloneSet(app, labels)
			if err != nil {
				return nil, err
			}
			raw, _ = json.Marshal(cloneset)
		} else {
			deployment, err := r.generateDeployment(app, labels)
			if err != nil {
				return nil, err
			}
			raw, _ = json.Marshal(deployment)
		}
	case rocketv1alpha1.Stateful:
		if _, ok := app.Annotations[constant.ExtendedResourceAnnotation]; ok {
			ests, err := r.generateStatefulSet(app, labels)
			if err != nil {
				return nil, err
			}
			raw, _ = json.Marshal(ests)
		} else {
			// TODO: 生成原生的statefulset
			sts, err := r.generateStatefulSet(app, labels)
			if err != nil {
				return nil, err
			}
			raw, _ = json.Marshal(sts)
		}
	case rocketv1alpha1.CronTask:
		cronjob, err := r.generateCronJob(app, labels)
		if err != nil {
			return nil, err
		}
		raw, _ = json.Marshal(cronjob)
	case rocketv1alpha1.Task:
		// TODO: 生成原生的job
		job, err := r.generateCronJob(app, labels)
		if err != nil {
			return nil, err
		}
		raw, _ = json.Marshal(job)
	}
	workload.Spec.Template.Raw = raw
	for _, v := range app.Spec.Traits {
		if traitOption, ok := trait.Traits[v.Kind]; ok {
			var err error
			workload, err = traitOption.Handler(&v, workload)
			if err != nil {
				return nil, err
			}
		}
	}
	return workload, nil
}

// generateDeployment generate deployment
func (r *workloadOption) generateDeployment(app *rocketv1alpha1.Application, label map[string]string) (
	*appsv1.Deployment, error) {
	// NOTE: deployment can not use volumeclaim
	podtemplate, _, _, err := r.generatePod(app)
	if err != nil {
		return nil, err
	}
	podtemplate.Labels = label
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: podtemplate,
		},
	}
	return deployment, nil
}

// generateCloneSet generate cloneset
func (r *workloadOption) generateCloneSet(app *rocketv1alpha1.Application, label map[string]string) (
	*kruiseappsv1alpha1.CloneSet, error) {
	podtemplate, diskSize, _, err := r.generatePod(app)
	if err != nil {
		return nil, err
	}
	podtemplate.Labels = label
	cloneset := &kruiseappsv1alpha1.CloneSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kruiseappsv1alpha1.SchemeGroupVersion.String(),
			Kind:       "CloneSet",
		},
		Spec: kruiseappsv1alpha1.CloneSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template:             podtemplate,
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{},
			UpdateStrategy: kruiseappsv1alpha1.CloneSetUpdateStrategy{
				// default InPlaceOnly
				Type: kruiseappsv1alpha1.InPlaceIfPossibleCloneSetUpdateStrategyType,
			},
		},
	}
	if !diskSize.IsZero() {
		cloneset.Spec.VolumeClaimTemplates = append(cloneset.Spec.VolumeClaimTemplates,
			v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: app.Name + "-data-",
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceStorage: diskSize,
						},
					},
				},
			})
	}
	return cloneset, nil
}

func (r *workloadOption) generateStatefulSet(app *rocketv1alpha1.Application, label map[string]string) (
	*kruiseappsv1beta1.StatefulSet, error) {
	return &kruiseappsv1beta1.StatefulSet{}, nil
}

func (r *workloadOption) generateCronJob(app *rocketv1alpha1.Application, label map[string]string) (
	*batchv1.CronJob, error) {
	// cronjob 忽略存储卷的创建
	podtemplate, _, temp, err := r.generatePod(app)
	if err != nil {
		return nil, err
	}
	// 在此处验证了模版的数量，如果不是1，直接报错
	if len(app.Spec.Templates) != 1 {
		return nil, fmt.Errorf("application templates must be one, got %d", len(app.Spec.Templates))
	}
	cmp := &rocketv1alpha1.Template{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: app.Namespace, Name: app.Spec.Templates[0].Name}, cmp)
	if err != nil {
		return nil, err
	}
	podtemplate.Spec.RestartPolicy = v1.RestartPolicy(temp.Spec.JobOptions.RestartPolicy)
	return &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "CronJob",
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   *temp.Spec.JobOptions.Schedule,
			Suspend:                    pointer.BoolPtr(temp.Spec.JobOptions.Suspend),
			SuccessfulJobsHistoryLimit: pointer.Int32Ptr(temp.Spec.JobOptions.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     pointer.Int32Ptr(temp.Spec.JobOptions.FailedJobsHistoryLimit),
			ConcurrencyPolicy:          batchv1.ConcurrencyPolicy(*temp.Spec.JobOptions.ConcurrencyPolicy),
			StartingDeadlineSeconds:    pointer.Int64Ptr(temp.Spec.JobOptions.StartingDeadlineSeconds),
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: pointer.Int32Ptr(temp.Spec.JobOptions.BackoffLimit),
					// TTLSecondsAfterFinished: pointer.Int32Ptr(temp.Spec.JobOptions.TTLSecondsAfterFinished),
					Template: podtemplate,
				},
			},
		},
	}, nil
}

func (r *workloadOption) generatePod(app *rocketv1alpha1.Application) (
	v1.PodTemplateSpec, resource.Quantity, *rocketv1alpha1.Template, error) {
	pod := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			HostAliases: []v1.HostAlias{},
			Containers:  []v1.Container{},
		},
	}
	diskSize := resource.NewQuantity(0, resource.BinarySI)
	temp := &rocketv1alpha1.Template{}
	for _, c := range app.Spec.Templates {
		cmp := &rocketv1alpha1.Template{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: app.Namespace, Name: c.Name}, cmp)
		if err != nil {
			return pod, *diskSize, nil, err
		}
		p := &parameterHandler{
			originalVariables:       app.Spec.Variables,
			originalParameterValues: c.ParameterValues,
			originalParameters:      cmp.Spec.Parameters,
		}
		parameters := p.renderParameter()
		v, err := r.generateVolume(diskSize, cmp.Spec.Containers, parameters)
		if err != nil {
			return pod, *diskSize, nil, err
		}
		diskSize = v
		containers, err := r.generateContainer(c.InstanceName, app.Spec.Environment,
			parameters, cmp, c.ImageDefine)
		if err != nil {
			return pod, *diskSize, nil, err
		}
		pod.Spec.Containers = append(pod.Spec.Containers, containers...)
		for idx, h := range cmp.Spec.HostAliases {
			// 将temp中的 hostalias 渲染
			cmp.Spec.HostAliases[idx].IP = stringRender(h.IP, parameters)
		}
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, cmp.Spec.HostAliases...)
		// 使用最后一个模板的数据
		temp = cmp
	}
	return pod, *diskSize, temp, nil
}

func (r *workloadOption) generateVolume(v *resource.Quantity, containers []rocketv1alpha1.Container,
	parameters map[string]string) (*resource.Quantity, error) {
	for _, container := range containers {
		if container.Resources != nil {
			if len(container.Resources.Volumes) != 0 {
				for _, vol := range container.Resources.Volumes {
					s, err := resource.ParseQuantity(vol.Size.Limits)
					if err != nil {
						return nil, err
					}
					if s.Value() > v.Value() {
						v = &s
					}
				}
			}
		}
	}
	return v, nil
}

func (r *workloadOption) generateContainer(instance, env string, parameters map[string]string,
	template *rocketv1alpha1.Template, images []rocketv1alpha1.ImageDefine) ([]v1.Container, error) {
	idict := map[string]string{}
	for _, img := range images {
		idict[img.ContainerName] = img.Image
	}
	containers := []v1.Container{}
	for _, c := range template.Spec.Containers {
		// container名称是app中定义的instance名称与temp中container名称组合而来,
		// 规则是 containerName-instanceName.
		containerName := fmt.Sprintf("%s-%s", c.Name, instance)
		container := v1.Container{
			Name:    containerName,
			Command: sliceRender(c.Command, parameters),
			Args:    sliceRender(c.Args, parameters),
			Ports:   c.Ports,
			// 设置默认的环境变量POD_IP和HOST_IP
			Env: []v1.EnvVar{
				{
					Name: "POD_IP",
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "status.podIP",
						},
					},
				},
				{
					Name: "HOST_IP",
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "status.hostIP",
						},
					},
				},
			},
			Lifecycle: lifecycleRender(c.Lifecycle, parameters),
			// TODO: probe不需要通过trait的方式来支持，应该通过渲染的方式设置
			LivenessProbe:  c.LivenessProbe,
			ReadinessProbe: c.ReadinessProbe,
			StartupProbe:   c.StartupProbe,
			VolumeMounts:   []v1.VolumeMount{},
		}
		if img, ok := idict[c.Name]; ok {
			container.Image = img
		} else {
			container.Image = c.Image
		}
		if c.Resources != nil {
			cpur, err := resourceRender(c.Resources.CPU.Requests, parameters)
			if err != nil {
				return nil, err
			}
			cpul, err := resourceRender(c.Resources.CPU.Limits, parameters)
			if err != nil {
				return nil, err
			}
			memr, err := resourceRender(c.Resources.Memory.Requests, parameters)
			if err != nil {
				return nil, err
			}
			meml, err := resourceRender(c.Resources.Memory.Limits, parameters)
			if err != nil {
				return nil, err
			}
			resources := v1.ResourceRequirements{
				Limits: v1.ResourceList(
					map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU:    *cpul,
						v1.ResourceMemory: *meml,
					}),
				Requests: v1.ResourceList(
					map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU:    *cpur,
						v1.ResourceMemory: *memr,
					}),
			}
			if c.Resources.EphemeralStorage.Limits != "" {
				evl, err := resource.ParseQuantity(c.Resources.EphemeralStorage.Limits)
				if err != nil {
					return nil, err
				}
				resources.Limits[v1.ResourceEphemeralStorage] = evl
			}
			if c.Resources.EphemeralStorage.Requests != "" {
				evr, err := resource.ParseQuantity(c.Resources.EphemeralStorage.Requests)
				if err != nil {
					return nil, err
				}
				resources.Requests[v1.ResourceEphemeralStorage] = evr
			}

			container.Resources = resources
			for _, v := range c.Resources.Volumes {
				volume := v1.VolumeMount{
					Name:     fmt.Sprintf("%s-%s", containerName, v.Name),
					ReadOnly: true,
				}
				if v.AccessMode == "RW" {
					volume.ReadOnly = false
				}
			}
		}
		for _, e := range envRender(c.Env, parameters) {
			env := v1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			}
			container.Env = append(container.Env, env)
		}
		containers = append(containers, container)
	}
	return containers, nil
}
