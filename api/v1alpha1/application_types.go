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

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Which cloud area the application in.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default=public
	// +kubebuilder:validation:Enum=public;private
	CloudArea string `json:"cloudArea,omitempty"`
	// prod, pre or test
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=dev;test;staging;prod
	Environment string `json:"environment,omitempty"`
	// The region of application
	// +required
	// +kubebuilder:validation:Required
	Regions []string `json:"regions,omitempty"`
	// The number of replicas of application pods, it will override the replicas in template.
	// +optional
	Replicas *int64 `json:"replicas,omitempty"`
	// Template must be the complete yaml that users want to distribute.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Template runtime.RawExtension `json:"template,omitempty"`
	// The ac this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	// The operation traits of this application.
	// +optional
	Traits []Trait `json:"traits,omitempty"`
}

type Trait struct {
	// The kind of trait CRD.
	// +required
	// +kubebuilder:validation:Required
	Kind string `json:"kind,omitempty"`
	// The version of trait CRD.
	// +optional
	Version string `json:"version,omitempty"`
	// The template of trait CRD.
	// +optional
	Template string `json:"template,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// the cluster of application
	// +optional
	Clusters []string `json:"clusters,omitempty"`
	// the phase of the ApplicationConfiguration
	// +kubebuilder:default=Pending
	// +kubebuilder:validation:Enum=Pending;Scheduling;Running
	Phase string `json:"phase,omitempty"`
	// application details
	// +optional
	ApplicationDetails *runtime.RawExtension `json:"applicationDetails,omitempty"`
	// cluster workload condition
	// +optional
	Conditions []ApplicationCondition `json:"conditions,omitempty"`
	// +optional
	Type string `json:"type,omitempty"`
}

type ApplicationConditionType string

const (
	// ApplicationConditionTypePending indicates that the application controller successed to create or delete the workload or edge trait.
	ApplicationConditionSuccessedScale ApplicationConditionType = "SuccessedScale"
	// ApplicationConditionTypePending indicates that the application controller successed to update the workload or edge trait.
	ApplicationConditionSuccessedUpdate ApplicationConditionType = "SuccessedUpdate"
	// ApplicationConditionTypePending indicates that the application controller unexpected to create or delete the workload or edge trait.
	ApplicationConditionUnexpectedScale ApplicationConditionType = "UnexpectedScale"
	// ApplicationConditionUnexpectedUpdate indicates that the application controller unexpected to update the workload or edge trait.
	ApplicationConditionUnexpectedUpdate ApplicationConditionType = "UnexpectedUpdate"
)

type ApplicationCondition struct {
	// type of condition in CamelCase or in foo.example.com/CamelCase.
	// ---
	// Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be
	// useful (see .node.status.conditions), the ability to deconflict is important.
	// The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)
	// +required
	// +kubebuilder:validation:Required
	Type ApplicationConditionType `json:"type"`
	// status of the condition, one of True, False, Unknown.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status metav1.ConditionStatus `json:"status"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// Producers of specific condition types may define expected values and meanings for this field,
	// and whether the values are considered a guaranteed API.
	// The value should be a CamelCase string.
	// This field may not be empty.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$`
	Reason string `json:"reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=app;apps,scope=Namespaced,categories=app;apps
// +kubebuilder:printcolumn:name="TYPE",priority=0,type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="AREA",priority=0,type=string,JSONPath=`.spec.cloudArea`
// +kubebuilder:printcolumn:name="REGION",priority=0,type=string,JSONPath=`.spec.regions[*]`
// +kubebuilder:printcolumn:name="EVNIRONMENT",priority=0,type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="CLUSTER",priority=0,type=string,JSONPath=`.status.clusters[*]`
// +kubebuilder:printcolumn:name="AGE",priority=0,type=date,JSONPath=`.metadata.creationTimestamp`

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
