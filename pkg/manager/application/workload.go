package application

import (
	"context"
	"fmt"

	rocketv1alpha1 "github.com/hex-techs/rocket/api/v1alpha1"
	"github.com/hex-techs/rocket/pkg/manager/application/trait"
	"github.com/hex-techs/rocket/pkg/util/constant"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
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

// 处理兼容数据
func (r *workloadOption) generateWorkload(kind rocketv1alpha1.WorkloadType, app *rocketv1alpha1.Application) (
	*rocketv1alpha1.Workload, error) {
	l := map[string]string{}
	if app.Labels != nil {
		l = app.Labels
	}
	l[constant.AppNameLabel] = app.Name
	// workload继承app的annotation和label，workload名称直接使用app的名称
	workload := &rocketv1alpha1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   app.Namespace,
			Name:        app.Name,
			Annotations: app.Annotations,
			Labels:      l,
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
			Template:    rocketv1alpha1.WorkloadTemplate{},
			Tolerations: app.Spec.Tolerations,
		},
	}
	// NOTE: The HostAliases is not set in the production environment
	switch kind {
	case rocketv1alpha1.Server:
		cloneset, err := r.generateCloneSet(app, l)
		if err != nil {
			return nil, err
		}
		if app.Spec.Environment != "prod" {
			cloneset.Template.Spec.HostAliases = []v1.HostAlias{}
		}
		workload.Spec.Template.CloneSetTemplate = cloneset
	case rocketv1alpha1.Worker:
		workload.Spec.Template.StatefulSetTemlate = r.generateStatefulSet(app)
	case rocketv1alpha1.Task:
		cj, err := r.generateCronJob(app, l)
		if err != nil {
			return nil, err
		}
		if app.Spec.Environment != "prod" {
			cj.JobTemplate.Spec.Template.Spec.HostAliases = []v1.HostAlias{}
		}
		workload.Spec.Template.CronJobTemplate = cj
	case rocketv1alpha1.SingletonTask:
	}
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

func (r *workloadOption) generateCloneSet(ac *rocketv1alpha1.Application, label map[string]string) (
	*kruiseappsv1alpha1.CloneSetSpec, error) {
	podtemplate, diskSize, err := r.generatePod(ac)
	if err != nil {
		return nil, err
	}
	podtemplate.Labels = label
	cloneset := &kruiseappsv1alpha1.CloneSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: label,
		},
		Template:             podtemplate,
		VolumeClaimTemplates: []v1.PersistentVolumeClaim{},
		UpdateStrategy: kruiseappsv1alpha1.CloneSetUpdateStrategy{
			// default InPlaceOnly
			Type: kruiseappsv1alpha1.InPlaceOnlyCloneSetUpdateStrategyType,
		},
	}
	if !diskSize.IsZero() {
		cloneset.VolumeClaimTemplates = append(cloneset.VolumeClaimTemplates,
			v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: ac.Name + "-data-",
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

func (r *workloadOption) generateStatefulSet(ac *rocketv1alpha1.Application) *kruiseappsv1alpha1.StatefulSetSpec {
	return &kruiseappsv1alpha1.StatefulSetSpec{}
}

func (r *workloadOption) generateCronJob(ac *rocketv1alpha1.Application, label map[string]string) (
	*batchv1.CronJobSpec, error) {
	aja := annotation{cronjobAnnotation: ac.Annotations}
	// cronjob 忽略存储卷的创建
	podtemplate, _, err := r.generatePod(ac)
	if err != nil {
		return nil, err
	}
	if len(ac.Spec.Templates) != 1 {
		return nil, fmt.Errorf("application templates must be one, got %d", len(ac.Spec.Templates))
	}
	cmp := &rocketv1alpha1.Template{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: ac.Namespace, Name: ac.Spec.Templates[0].Name}, cmp)
	if err != nil {
		return nil, err
	}
	podtemplate.Spec.RestartPolicy = aja.getRestartPolicy(cmp)
	return &batchv1.CronJobSpec{
		Schedule:                   aja.getSchedule(cmp),
		Suspend:                    aja.getSuspend(cmp),
		SuccessfulJobsHistoryLimit: aja.getSuccessfulJobsHistoryLimit(cmp),
		FailedJobsHistoryLimit:     aja.getFailedJobsHistoryLimit(cmp),
		ConcurrencyPolicy:          aja.getConcurrencyPolicy(cmp),
		StartingDeadlineSeconds:    aja.getStartingDeadlineSeconds(cmp),
		JobTemplate: batchv1.JobTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: label,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit:            aja.getBackoffLimit(cmp),
				TTLSecondsAfterFinished: aja.getTTLSecondsAfterFinished(cmp),
				Template:                podtemplate,
			},
		},
	}, nil
}

func (r *workloadOption) generatePod(ac *rocketv1alpha1.Application) (
	v1.PodTemplateSpec, resource.Quantity, error) {
	pod := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			HostAliases: []v1.HostAlias{},
			Containers:  []v1.Container{},
		},
	}
	diskSize := resource.NewQuantity(0, resource.BinarySI)
	for _, c := range ac.Spec.Templates {
		cmp := &rocketv1alpha1.Template{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: ac.Namespace, Name: c.Name}, cmp)
		if err != nil {
			return pod, *diskSize, err
		}
		p := &parameterHandler{
			originalVariables:       ac.Spec.Variables,
			originalParameterValues: c.ParameterValues,
			originalParameters:      cmp.Spec.Parameters,
		}
		parameters := p.renderParameter()
		v, err := r.generateVolume(diskSize, cmp.Spec.Containers, parameters)
		if err != nil {
			return pod, *diskSize, err
		}
		diskSize = v
		containers, err := r.generateContainer(c.InstanceName, ac.Spec.Environment,
			parameters, cmp, c.ImageDefine)
		if err != nil {
			return pod, *diskSize, err
		}
		pod.Spec.Containers = append(pod.Spec.Containers, containers...)
		for idx, h := range cmp.Spec.HostAliases {
			// 将temp中的 hostalias 渲染
			cmp.Spec.HostAliases[idx].IP = stringRender(h.IP, parameters)
		}
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, cmp.Spec.HostAliases...)
	}
	return pod, *diskSize, nil
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
