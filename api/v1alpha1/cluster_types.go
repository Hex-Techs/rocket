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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// ID is the uid for this cluster.
	// +required
	// +kubebuilder:validation:Required
	ID string `json:"id,omitempty"`
	// Region is the region of the cluster, e.g. ap-beijing.
	// Now, only support ap-beijing and ap-guangzhou.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ap-beijing;ap-guangzhou
	Region string `json:"region,omitempty"`
	// public or private
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ded;pub
	Area string `json:"area,omitempty"`
	// prod, pre or test
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=pd;pre;tst
	Environment string `json:"environment,omitempty"`
	// the cluster connect api server
	// +optional
	APIServer string `json:"apiServer,omitempty"`
	// +optional
	CAData []byte `json:"caData,omitempty"`
	// +optional
	CertData []byte `json:"certData,omitempty"`
	// +optional
	KeyData []byte `json:"keyData,omitempty"`
	// serviceaccount token to connect with agent clustr
	// +optional
	Token []byte `json:"token,omitempty"`
	// Taints is the taint of the cluster
	// +optional
	Taints []v1.Taint `json:"taints,omitempty"`
}

// ClusterState the cluster which register's state
type ClusterState string

const (
	// the cluster is wait for approve
	Pending ClusterState = "Pending"
	// the cluster is approved
	Approve ClusterState = "Approved"
	// the cluster is rejected
	Reject ClusterState = "Rejected"
	// when the agent is not send heartbeat
	Offline ClusterState = "Offline"
)

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// the state of the cluster
	// +kubebuilder:default=Pending
	// +kubebuilder:validation:Enum=Pending;Approved;Rejected;Offline
	State ClusterState `json:"state,omitempty"`
	// the last time heartbeat from agent
	LastKeepAliveTime metav1.Time `json:"lastKeepAliveTime,omitempty"`
	// Cluster ID, it is tke id when cluster in tencent cloud, the other is generated id
	ID string `json:"id,omitempty"`
	// kubernetes version
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// ready node count of the kubernetes cluster
	ReadyNodeCount *int `json:"readyNodeCount,omitempty"`
	// unhealth node count of the kubernetes cluster
	UnhealthNodeCount *int `json:"unhealthNodeCount,omitempty"`
	// allocatable the allocatable resources of the cluster
	Allocatable v1.ResourceList `json:"allocatable,omitempty"`
	// capacity the capacity resources of the cluster
	Capacity v1.ResourceList `json:"capacity,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="AREA",priority=0,type=string,JSONPath=`.spec.area`
// +kubebuilder:printcolumn:name="REGION",priority=0,type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="EVNIRONMENT",priority=0,type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="STATE",priority=0,type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="AGE",priority=0,type=date,JSONPath=`.metadata.creationTimestamp`

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
