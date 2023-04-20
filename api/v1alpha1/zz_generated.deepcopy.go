//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	appsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	"github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Application) DeepCopyInto(out *Application) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Application.
func (in *Application) DeepCopy() *Application {
	if in == nil {
		return nil
	}
	out := new(Application)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Application) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationList) DeepCopyInto(out *ApplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Application, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationList.
func (in *ApplicationList) DeepCopy() *ApplicationList {
	if in == nil {
		return nil
	}
	out := new(ApplicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationSpec) DeepCopyInto(out *ApplicationSpec) {
	*out = *in
	if in.Regions != nil {
		in, out := &in.Regions, &out.Regions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Variables != nil {
		in, out := &in.Variables, &out.Variables
		*out = make([]Variable, len(*in))
		copy(*out, *in)
	}
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]ApplicationTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Traits != nil {
		in, out := &in.Traits, &out.Traits
		*out = make([]Trait, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationSpec.
func (in *ApplicationSpec) DeepCopy() *ApplicationSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationStatus) DeepCopyInto(out *ApplicationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TraitCondition != nil {
		in, out := &in.TraitCondition, &out.TraitCondition
		*out = new(metav1.Condition)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationStatus.
func (in *ApplicationStatus) DeepCopy() *ApplicationStatus {
	if in == nil {
		return nil
	}
	out := new(ApplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationTemplate) DeepCopyInto(out *ApplicationTemplate) {
	*out = *in
	if in.ImageDefine != nil {
		in, out := &in.ImageDefine, &out.ImageDefine
		*out = make([]ImageDefine, len(*in))
		copy(*out, *in)
	}
	if in.ParameterValues != nil {
		in, out := &in.ParameterValues, &out.ParameterValues
		*out = make([]ParameterValue, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationTemplate.
func (in *ApplicationTemplate) DeepCopy() *ApplicationTemplate {
	if in == nil {
		return nil
	}
	out := new(ApplicationTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplyScope) DeepCopyInto(out *ApplyScope) {
	*out = *in
	if in.CloudAreas != nil {
		in, out := &in.CloudAreas, &out.CloudAreas
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplyScope.
func (in *ApplyScope) DeepCopy() *ApplyScope {
	if in == nil {
		return nil
	}
	out := new(ApplyScope)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterList) DeepCopyInto(out *ClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterList.
func (in *ClusterList) DeepCopy() *ClusterList {
	if in == nil {
		return nil
	}
	out := new(ClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *in
	if in.CAData != nil {
		in, out := &in.CAData, &out.CAData
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.CertData != nil {
		in, out := &in.CertData, &out.CertData
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.KeyData != nil {
		in, out := &in.KeyData, &out.KeyData
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.Taints != nil {
		in, out := &in.Taints, &out.Taints
		*out = make([]v1.Taint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSpec.
func (in *ClusterSpec) DeepCopy() *ClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *in
	in.LastKeepAliveTime.DeepCopyInto(&out.LastKeepAliveTime)
	if in.ReadyNodeCount != nil {
		in, out := &in.ReadyNodeCount, &out.ReadyNodeCount
		*out = new(int)
		**out = **in
	}
	if in.UnhealthNodeCount != nil {
		in, out := &in.UnhealthNodeCount, &out.UnhealthNodeCount
		*out = new(int)
		**out = **in
	}
	if in.Allocatable != nil {
		in, out := &in.Allocatable, &out.Allocatable
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	if in.Capacity != nil {
		in, out := &in.Capacity, &out.Capacity
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatus.
func (in *ClusterStatus) DeepCopy() *ClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Container) DeepCopyInto(out *Container) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ContainerPort, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]Env, len(*in))
		copy(*out, *in)
	}
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.StartupProbe != nil {
		in, out := &in.StartupProbe, &out.StartupProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(ContainerResource)
		(*in).DeepCopyInto(*out)
	}
	if in.Lifecycle != nil {
		in, out := &in.Lifecycle, &out.Lifecycle
		*out = new(v1.Lifecycle)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Container.
func (in *Container) DeepCopy() *Container {
	if in == nil {
		return nil
	}
	out := new(Container)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerResource) DeepCopyInto(out *ContainerResource) {
	*out = *in
	out.CPU = in.CPU
	out.Memory = in.Memory
	out.EphemeralStorage = in.EphemeralStorage
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]ContainerVolume, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerResource.
func (in *ContainerResource) DeepCopy() *ContainerResource {
	if in == nil {
		return nil
	}
	out := new(ContainerResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerVolume) DeepCopyInto(out *ContainerVolume) {
	*out = *in
	out.Size = in.Size
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerVolume.
func (in *ContainerVolume) DeepCopy() *ContainerVolume {
	if in == nil {
		return nil
	}
	out := new(ContainerVolume)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Distribution) DeepCopyInto(out *Distribution) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Distribution.
func (in *Distribution) DeepCopy() *Distribution {
	if in == nil {
		return nil
	}
	out := new(Distribution)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Distribution) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionCluster) DeepCopyInto(out *DistributionCluster) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionCluster.
func (in *DistributionCluster) DeepCopy() *DistributionCluster {
	if in == nil {
		return nil
	}
	out := new(DistributionCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionCondition) DeepCopyInto(out *DistributionCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionCondition.
func (in *DistributionCondition) DeepCopy() *DistributionCondition {
	if in == nil {
		return nil
	}
	out := new(DistributionCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionList) DeepCopyInto(out *DistributionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Distribution, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionList.
func (in *DistributionList) DeepCopy() *DistributionList {
	if in == nil {
		return nil
	}
	out := new(DistributionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DistributionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionSpec) DeepCopyInto(out *DistributionSpec) {
	*out = *in
	in.Resource.DeepCopyInto(&out.Resource)
	in.Targets.DeepCopyInto(&out.Targets)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionSpec.
func (in *DistributionSpec) DeepCopy() *DistributionSpec {
	if in == nil {
		return nil
	}
	out := new(DistributionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionStatus) DeepCopyInto(out *DistributionStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(map[string]DistributionCondition, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionStatus.
func (in *DistributionStatus) DeepCopy() *DistributionStatus {
	if in == nil {
		return nil
	}
	out := new(DistributionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionTargetClusters) DeepCopyInto(out *DistributionTargetClusters) {
	*out = *in
	if in.List != nil {
		in, out := &in.List, &out.List
		*out = make([]DistributionCluster, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionTargetClusters.
func (in *DistributionTargetClusters) DeepCopy() *DistributionTargetClusters {
	if in == nil {
		return nil
	}
	out := new(DistributionTargetClusters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributionTargets) DeepCopyInto(out *DistributionTargets) {
	*out = *in
	in.IncludedClusters.DeepCopyInto(&out.IncludedClusters)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributionTargets.
func (in *DistributionTargets) DeepCopy() *DistributionTargets {
	if in == nil {
		return nil
	}
	out := new(DistributionTargets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Env) DeepCopyInto(out *Env) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Env.
func (in *Env) DeepCopy() *Env {
	if in == nil {
		return nil
	}
	out := new(Env)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageDefine) DeepCopyInto(out *ImageDefine) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageDefine.
func (in *ImageDefine) DeepCopy() *ImageDefine {
	if in == nil {
		return nil
	}
	out := new(ImageDefine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobOptions) DeepCopyInto(out *JobOptions) {
	*out = *in
	if in.Schedule != nil {
		in, out := &in.Schedule, &out.Schedule
		*out = new(string)
		**out = **in
	}
	if in.ConcurrencyPolicy != nil {
		in, out := &in.ConcurrencyPolicy, &out.ConcurrencyPolicy
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobOptions.
func (in *JobOptions) DeepCopy() *JobOptions {
	if in == nil {
		return nil
	}
	out := new(JobOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Parameter) DeepCopyInto(out *Parameter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Parameter.
func (in *Parameter) DeepCopy() *Parameter {
	if in == nil {
		return nil
	}
	out := new(Parameter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParameterValue) DeepCopyInto(out *ParameterValue) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParameterValue.
func (in *ParameterValue) DeepCopy() *ParameterValue {
	if in == nil {
		return nil
	}
	out := new(ParameterValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceQuantity) DeepCopyInto(out *ResourceQuantity) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceQuantity.
func (in *ResourceQuantity) DeepCopy() *ResourceQuantity {
	if in == nil {
		return nil
	}
	out := new(ResourceQuantity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Template) DeepCopyInto(out *Template) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Template.
func (in *Template) DeepCopy() *Template {
	if in == nil {
		return nil
	}
	out := new(Template)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Template) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplateList) DeepCopyInto(out *TemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Template, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplateList.
func (in *TemplateList) DeepCopy() *TemplateList {
	if in == nil {
		return nil
	}
	out := new(TemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplateSpec) DeepCopyInto(out *TemplateSpec) {
	*out = *in
	in.ApplyScope.DeepCopyInto(&out.ApplyScope)
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		copy(*out, *in)
	}
	if in.HostAliases != nil {
		in, out := &in.HostAliases, &out.HostAliases
		*out = make([]v1.HostAlias, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = make([]Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.JobOptions != nil {
		in, out := &in.JobOptions, &out.JobOptions
		*out = new(JobOptions)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplateSpec.
func (in *TemplateSpec) DeepCopy() *TemplateSpec {
	if in == nil {
		return nil
	}
	out := new(TemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplateStatus) DeepCopyInto(out *TemplateStatus) {
	*out = *in
	if in.UsedBy != nil {
		in, out := &in.UsedBy, &out.UsedBy
		*out = make([]UsedBy, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplateStatus.
func (in *TemplateStatus) DeepCopy() *TemplateStatus {
	if in == nil {
		return nil
	}
	out := new(TemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Trait) DeepCopyInto(out *Trait) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Trait.
func (in *Trait) DeepCopy() *Trait {
	if in == nil {
		return nil
	}
	out := new(Trait)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UsedBy) DeepCopyInto(out *UsedBy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UsedBy.
func (in *UsedBy) DeepCopy() *UsedBy {
	if in == nil {
		return nil
	}
	out := new(UsedBy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Variable) DeepCopyInto(out *Variable) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Variable.
func (in *Variable) DeepCopy() *Variable {
	if in == nil {
		return nil
	}
	out := new(Variable)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Workload) DeepCopyInto(out *Workload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Workload.
func (in *Workload) DeepCopy() *Workload {
	if in == nil {
		return nil
	}
	out := new(Workload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Workload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadList) DeepCopyInto(out *WorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Workload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadList.
func (in *WorkloadList) DeepCopy() *WorkloadList {
	if in == nil {
		return nil
	}
	out := new(WorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadSpec) DeepCopyInto(out *WorkloadSpec) {
	*out = *in
	if in.Regions != nil {
		in, out := &in.Regions, &out.Regions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Template.DeepCopyInto(&out.Template)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadSpec.
func (in *WorkloadSpec) DeepCopy() *WorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(WorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadStatus) DeepCopyInto(out *WorkloadStatus) {
	*out = *in
	if in.Clusters != nil {
		in, out := &in.Clusters, &out.Clusters
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(map[string]metav1.Condition, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.DeploymentStatus != nil {
		in, out := &in.DeploymentStatus, &out.DeploymentStatus
		*out = new(appsv1.DeploymentStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.ClonSetStatus != nil {
		in, out := &in.ClonSetStatus, &out.ClonSetStatus
		*out = new(appsv1alpha1.CloneSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.StatefulSetStatus != nil {
		in, out := &in.StatefulSetStatus, &out.StatefulSetStatus
		*out = new(appsv1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.ExtendStatefulSetStatus != nil {
		in, out := &in.ExtendStatefulSetStatus, &out.ExtendStatefulSetStatus
		*out = new(appsv1alpha1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.CronjobStatus != nil {
		in, out := &in.CronjobStatus, &out.CronjobStatus
		*out = new(batchv1.CronJobStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.JobStatus != nil {
		in, out := &in.JobStatus, &out.JobStatus
		*out = new(batchv1.JobStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadStatus.
func (in *WorkloadStatus) DeepCopy() *WorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(WorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadTemplate) DeepCopyInto(out *WorkloadTemplate) {
	*out = *in
	if in.DeploymentTemplate != nil {
		in, out := &in.DeploymentTemplate, &out.DeploymentTemplate
		*out = new(appsv1.DeploymentSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.StatefulSetTemlate != nil {
		in, out := &in.StatefulSetTemlate, &out.StatefulSetTemlate
		*out = new(appsv1.StatefulSetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CloneSetTemplate != nil {
		in, out := &in.CloneSetTemplate, &out.CloneSetTemplate
		*out = new(appsv1alpha1.CloneSetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ExtendStatefulSetTemlate != nil {
		in, out := &in.ExtendStatefulSetTemlate, &out.ExtendStatefulSetTemlate
		*out = new(v1beta1.StatefulSetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CronJobTemplate != nil {
		in, out := &in.CronJobTemplate, &out.CronJobTemplate
		*out = new(batchv1.CronJobSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.JobTemplate != nil {
		in, out := &in.JobTemplate, &out.JobTemplate
		*out = new(batchv1.JobTemplateSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadTemplate.
func (in *WorkloadTemplate) DeepCopy() *WorkloadTemplate {
	if in == nil {
		return nil
	}
	out := new(WorkloadTemplate)
	in.DeepCopyInto(out)
	return out
}
