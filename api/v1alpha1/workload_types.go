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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// WorkloadSpec defines the desired state of Workload
type WorkloadSpec struct {
	// +optional
	Regions []string `json:"regions,omitempty"`
	// Template must be the complete yaml that users want to distribute.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Template runtime.RawExtension `json:"template,omitempty"`
	// The workload this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// WorkloadStatus defines the observed state of Workload
type WorkloadStatus struct {
	// +optional
	Clusters []string `json:"clusters,omitempty"`
	// the phase of the ApplicationConfiguration
	// +kubebuilder:default=Pending
	// +kubebuilder:validation:Enum=Pending;Scheduling;Running
	Phase string `json:"phase,omitempty"`
	// workload details
	// +optional
	WorkloadDetails *runtime.RawExtension `json:"workloadDetails,omitempty"`
	// cluster workload condition
	// +optional
	Conditions map[string]metav1.Condition `json:"conditions,omitempty"`
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
