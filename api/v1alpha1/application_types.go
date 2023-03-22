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
	// The variables of this application.
	// +optional
	Variables []Variable `json:"variables,omitempty"`
	// The templates were used by this application.
	// +optional
	Templates []ApplicationTemplate `json:"templates,omitempty"`
	// The ac this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	// The operation traits of this application.
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

type Variable struct {
	// The name of Variable
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	Value string `json:"value,omitempty"`
}

// ApplicationTemplate is the template used by.
type ApplicationTemplate struct {
	// The template name.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The generate container name for workload,
	// this is based on container, if not set.
	// +optional
	InstanceName string `json:"instanceName,omitempty"`
	// How to set this to owerwrite the default image.
	// +optional
	ImageDefine []ImageDefine `json:"imageDefine,omitempty"`
	// +optional
	ParameterValues []ParameterValue `json:"parameterValues,omitempty"`
}

type ImageDefine struct {
	// The container name of template conatiner
	// +required
	// +kubebuilder:validation:Required
	ContainerName string `json:"containerName,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	Image string `json:"image,omitempty"`
}

// ParameterValue the parameter of this template and overwrite.
type ParameterValue struct {
	// parameter name.
	// +required
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The value of this parameter, support '[fromVariable(message)]'
	// or metadata value.
	// +required
	// +kubebuilder:validation:Required
	Value string `json:"value,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// The result of the applicationconfiguration validation
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	WorkloadType WorkloadType `json:"workloadType,omitempty"`
	// +optional
	TraitCondition *metav1.Condition `json:"traitCondition,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=app;apps,scope=Namespaced,categories=app;apps
// +kubebuilder:printcolumn:name="WORKLOADTYPE",priority=0,type=string,JSONPath=`.status.workloadType`
// +kubebuilder:printcolumn:name="AREA",priority=0,type=string,JSONPath=`.spec.cloudArea`
// +kubebuilder:printcolumn:name="REGION",priority=0,type=string,JSONPath=`.spec.regions[*]`
// +kubebuilder:printcolumn:name="EVNIRONMENT",priority=0,type=string,JSONPath=`.spec.environment`
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
