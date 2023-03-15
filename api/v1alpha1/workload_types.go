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
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadSpec defines the desired state of Workload
type WorkloadSpec struct {
	// +optional
	Regions []string `json:"regions,omitempty"`
	// +optional
	Template WorkloadTemplate `json:"tempate,omitempty"`
	// The workload this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

type WorkloadTemplate struct {
	// +optional
	CloneSetTemplate *kruiseappsv1alpha1.CloneSetSpec `json:"clonesetTemplate,omitempty"`
	// +optional
	StatefulSetTemlate *kruiseappsv1alpha1.StatefulSetSpec `json:"statefulsetTemplate,omitempty"`
	// +optional
	CronJobTemplate *batchv1.CronJobSpec `json:"cronjobTemplate,omitempty"`
	// +optional
	JobTemplate *batchv1.JobTemplateSpec `json:"jobTemplate,omitempty"`
}

// WorkloadStatus defines the observed state of Workload
type WorkloadStatus struct {
	// +optional
	Clusters []string `json:"clusters,omitempty"`
	// the phase of the ApplicationConfiguration
	// +kubebuilder:default=Pending
	// +kubebuilder:validation:Enum=Pending;Scheduling;Running
	Phase string `json:"phase,omitempty"`
	// cluster workload condition
	// +optional
	Conditions map[string]metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ClonSetStatus *kruiseappsv1alpha1.CloneSetStatus `json:"clonesetStatus,omitempty"`
	// +optional
	StatefulSetStatus *kruiseappsv1alpha1.StatefulSetStatus `json:"statefulsetStatus,omitempty"`
	// +optional
	CronjobStatus *batchv1.CronJobStatus `json:"cronjobStatus,omitempty"`
	// +optional
	JobStatus *batchv1.JobStatus `json:"jobStatus,omitempty"`
	// +optional
	Type WorkloadType `json:"type,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=wl;wls,scope=Namespaced
// +kubebuilder:printcolumn:name="OWNER",priority=0,type=string,JSONPath=`.metadata.ownerReferences[*].name`
// +kubebuilder:printcolumn:name="REGION",priority=0,type=string,JSONPath=`.spec.regions[*]`
// +kubebuilder:printcolumn:name="CLUSTER",priority=0,type=string,JSONPath=`.status.clusters[*]`
// +kubebuilder:printcolumn:name="TYPE",priority=0,type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="PHASE",priority=0,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",priority=0,type=date,JSONPath=`.metadata.creationTimestamp`

// Workload is the Schema for the workloads API
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkloadList contains a list of Workload
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
