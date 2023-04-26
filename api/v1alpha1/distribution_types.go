package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DistributionSpec defines the desired state of Distribution.
type DistributionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// whether to enable the resource distribution
	// +optional
	Deploy bool `json:"deploy"`

	// Resource must be the complete yaml that users want to distribute.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Resource runtime.RawExtension `json:"resource"`
	// Targets defines the clusters that users want to distribute to.
	Targets DistributionTargets `json:"targets"`
}

// DistributionTargets defines the targets of Resource.
// Four options are provided to select target clusters.
type DistributionTargets struct {
	// If IncludedClusters is not empty, Resource will be distributed to the listed clusters.
	// +optional
	IncludedClusters DistributionTargetClusters `json:"includedClusters,omitempty"`
	// All will distribute Resource to all clusters.
	// +optional
	All *bool `json:"all,omitempty"`
}

type DistributionTargetClusters struct {
	/*
		// TODO: support regular expression in the future
		// +optional
		Pattern string `json:"pattern,omitempty"`
	*/
	// +optional
	List []DistributionCluster `json:"list,omitempty"`
}

// DistributionCluster contains a cluster name
type DistributionCluster struct {
	// Cluster name
	Name string `json:"name,omitempty"`
}

// DistributionStatus defines the observed state of Distribution.
// DistributionStatus is recorded by kruise, users' modification is invalid and meaningless.
type DistributionStatus struct {
	// Conditions describe the condition when Resource creating, updating and deleting.
	Conditions map[string]DistributionCondition `json:"conditions,omitempty"`
}

// DistributionCondition allows a row to be marked with additional information.
type DistributionCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason describe human readable message indicating details about last transition.
	Reason string `json:"reason,omitempty"`
	// Message describe human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",priority=0,type=date,JSONPath=`.metadata.creationTimestamp`

// Distribution is the Schema for the resourcedistributions API.
type Distribution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DistributionSpec   `json:"spec,omitempty"`
	Status DistributionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DistributionList contains a list of Distribution.
type DistributionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Distribution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Distribution{}, &DistributionList{})
}
