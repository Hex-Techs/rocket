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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkloadType string

const (
	// Long-Running workload type, it is 'CloneSet'.
	Stateless WorkloadType = "stateless"
	// Long-Running workload type, it is 'StatefulSet'.
	Stateful WorkloadType = "stateful"
	// One-Time workload type and periodic execution, it is 'CronJob'.
	CronTask WorkloadType = "cronTask"
	// One-Time workload type and execute only once, it is 'Job'.
	Task WorkloadType = "task"
)

// TemplateSpec defines the desired state of Template
type TemplateSpec struct {
	// The Template workloadType.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=stateless;stateful;cronTask;task
	WorkloadType WorkloadType `json:"workloadType,omitempty"`
	// The Template working range.
	// +kubebuilder:validation:Required
	ApplyScope ApplyScope `json:"applyScope,omitempty"`
	// The parameters of this template.
	// +optional
	Parameters []Parameter `json:"parameters,omitempty"`
	// Hholds the mapping between IP and hostnames that will be injected as an entry in the
	// pod's hosts file.
	// +optional
	HostAliases []v1.HostAlias `json:"hostAliases,omitempty"`
	// The container setting for this template.
	// +optional
	Containers []Container `json:"containers,omitempty"`
	// JobOptions is the options for job.
	// +optional
	JobOptions *JobOptions `json:"jobOptions,omitempty"`
}

// ApplyScope is a working range of the template
type ApplyScope struct {
	// Which cloud area the template can be worked.
	// +required
	// +kubebuilder:validation:Required
	CloudAreas []string `json:"cloudAreas,omitempty"`
	// Allow multiple references
	// +kubebuilder:default=true
	AllowOverlap bool `json:"allowOverlap,omitempty"`
}

// Parameter is a description of a parameter for a template
type Parameter struct {
	// The name of this parameter.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The parameter type.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=number;string;boolean
	Type string `json:"type,omitempty"`
	// A description of this parameter.
	// +optional
	Description string `json:"description,omitempty"`
	// This parameter is required.
	// +optional
	Required bool `json:"required,omitempty"`
	// The default value of this parameter.
	// +optional
	Default string `json:"default,omitempty"`
}

// Container is the template container description.
type Container struct {
	// Name is the container name.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The container image.
	// +required
	// +kubebuilder:validation:Required
	Image string `json:"image,omitempty"`
	// Start this container use this command.
	// +optional
	Command []string `json:"command,omitempty"`
	// The parameter of command.
	// fromParam syntax
	// +optional
	Args []string `json:"args,omitempty"`
	// The process working on which port.
	// +optional
	Ports []v1.ContainerPort `json:"ports,omitempty"`
	// The container environment variables.
	// fromParam syntax
	// +optional
	Env []Env `json:"env,omitempty"`
	// Use this probe judge the container is live.
	// +optional
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// Use this probe judge the container can be accept request.
	// +optional
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// Use this probe judge this container start success, this checker is
	// success, livenessProbe and readinessProbe will work.
	// +optional
	StartupProbe *v1.Probe `json:"startupProbe,omitempty"`
	// The container required resources
	// +required
	// +kubebuilder:validation:Required
	Resources *ContainerResource `json:"resources,omitempty"`
	// Actions that the management system should take in response to container lifecycle events.
	// +optional
	Lifecycle *v1.Lifecycle `json:"lifecycle,omitempty"`
}

// Env is the environment of this container.
type Env struct {
	// The name of this environment.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The value of this environment.
	// +optional
	Value string `json:"value,omitempty"`
	// Only use value or fromParam.
	// +optional
	FromParam string `json:"fromParam,omitempty"`
}

// ContainerResource is the resource of a container
type ContainerResource struct {
	// The container limits cpu
	// +required
	// +kubebuilder:validation:Required
	CPU ResourceQuantity `json:"cpu,omitempty"`
	// The container limits memory
	// +required
	// +kubebuilder:validation:Required
	Memory ResourceQuantity `json:"memory,omitempty"`
	// The ephemeral-storage limits for container
	// +optional
	EphemeralStorage ResourceQuantity `json:"ephemeralStorage,omitempty"`
	// The container use pvc
	// +optional
	Volumes []ContainerVolume `json:"volumes,omitempty"`
}

// ResourceQuantity the resource of container
type ResourceQuantity struct {
	// The limit resource, dev
	// +optional
	Limits string `json:"limits,omitempty"`
	// +optional
	Requests string `json:"requests,omitempty"`
}

// JobOptions is the job option for cron
// only work for cronTask and task
type JobOptions struct {
	// schedule is the cron schedule
	// If not specified, then it is a task.
	// +optional
	Schedule *string `json:"schedule,omitempty"`
	// suspend is the cron suspend
	// +optional
	Suspend bool `json:"suspend,omitempty"`
	// concurrencyPolicy is the cron concurrencyPolicy
	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	ConcurrencyPolicy *string `json:"concurrencyPolicy,omitempty"`
	// successfulJobsHistoryLimit is the cron successfulJobsHistoryLimit
	// +kubebuilder:default=3
	SuccessfulJobsHistoryLimit int32 `json:"successfulJobsHistoryLimit,omitempty"`
	// failedJobsHistoryLimit is the cron failedJobsHistoryLimit
	// +kubebuilder:default=1
	FailedJobsHistoryLimit int32 `json:"failedJobsHistoryLimit,omitempty"`
	// startingDeadlineSeconds is the cron startingDeadlineSeconds
	// +kubebuilder:default=30
	StartingDeadlineSeconds int64 `json:"startingDeadlineSeconds,omitempty"`
	// Backoff limit
	// +kubebuilder:default=0
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
	// Restart policy
	// +kubebuilder:default=Never
	RestartPolicy string `json:"restartPolicy,omitempty"`
}

// ContainerVolume the container volume
type ContainerVolume struct {
	// Name is the volume name.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Which path in container will mount.
	// +required
	// +kubebuilder:validation:Required
	MountPath string `json:"mountPath,omitempty"`
	// The path access mode.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default=RW
	// +kubebuilder:validation:Enum=RO;RW
	AccessMode string `json:"accessMode,omitempty"`
	// The size of this volume.
	// +required
	// +kubebuilder:validation:Required
	Size ResourceQuantity `json:"size,omitempty"`
}

// WorkloadUsedBy which workload use template
type UsedBy struct {
	// The name of application
	Name string `json:"name,omitempty"`
	// The uid of application
	UID string `json:"uid,omitempty"`
}

// TemplateStatus defines the observed state of Template
type TemplateStatus struct {
	// Whether it can be used again.
	// +required
	// +kubebuilder:validation:Required
	UsedAgain metav1.ConditionStatus `json:"usedAgain,omitempty"`
	// The result of the template validation
	// +optional
	UsedBy []UsedBy `json:"usedBy,omitempty"`
	// The number of workload used.
	// +kubebuilder:default=0
	NumberOfWorkload int `json:"numberOfWorkload,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=temp;temps,scope=Namespaced
// +kubebuilder:printcolumn:name="WORKLOADTYPE",priority=0,type=string,JSONPath=`.spec.workloadType`
// +kubebuilder:printcolumn:name="CLOUDAREA",priority=0,type=string,JSONPath=`.spec.applyScope.cloudAreas[*]`
// +kubebuilder:printcolumn:name="ALLOWOVERLAP",priority=0,type=string,JSONPath=`.spec.applyScope.allowOverlap`
// +kubebuilder:printcolumn:name="USENUMBER",priority=0,type=integer,JSONPath=`.status.numberOfWorkload`
// +kubebuilder:printcolumn:name="AGE",priority=0,type=date,JSONPath=`.metadata.creationTimestamp`

// Template is the Schema for the templates API
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateSpec   `json:"spec,omitempty"`
	Status TemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TemplateList contains a list of Template
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}
