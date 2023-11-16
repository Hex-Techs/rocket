package application

import v1 "k8s.io/api/core/v1"

type WorkloadType string

const (
	DeploymentType WorkloadType = "Deployment"
	CronJobType    WorkloadType = "CronJob"
	CloneSetType   WorkloadType = "CloneSet"
)

type Workload struct {
	Type     WorkloadType `json:"type,omitempty" binding:"required"`
	Sidecars []string     `json:"sidecars,omitempty"`
	// Name is the name of the container
	Name string `json:"name,omitempty" binding:"required"`
	// Image is the image of the container
	Image string `json:"image,omitempty" binding:"required"`
	// Command is the command of the container
	Command []string `json:"command,omitempty"`
	// Args is the args of the container
	Args []string `json:"args,omitempty"`
	// Env is the env of the container
	Env []v1.EnvVar `json:"env,omitempty"`
	// Ports is the ports of the container
	Ports []v1.ContainerPort `json:"ports,omitempty"`
	// Resources is the resources of the container
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// WorkingDir is the workingDir of the container
	WorkingDir string `json:"workingDir,omitempty"`
}
