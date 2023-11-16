package cluster

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hex-techs/rocket/pkg/models"
	"gorm.io/gorm"
)

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

// Cluster is the Schema for the clusters API
type Cluster struct {
	models.Base
	// Name is the name of the cluster
	Name string `gorm:"size:128,not null,index,unique" json:"name,omitempty" binding:"required"`
	// Description is the description of the cluster
	Description string `gorm:"size:1024" json:"description,omitempty"`
	// Region is the region of the cluster
	Region string `gorm:"size:64,not null" json:"region,omitempty" binding:"required"`
	// the cloud area of the cluster, e.g. private or public
	CloudArea string `gorm:"column:cloud_area,size:64,not null" json:"cloudArea,omitempty" binding:"required"`
	// the cluster connect api server
	APIServer string `gorm:"size:256" json:"apiServer,omitempty"`
	// ca data of the cluster
	CAData string `gorm:"type:text" json:"caData,omitempty"`
	// cert data of the cluster
	CertData string `gorm:"type:text" json:"certData,omitempty"`
	// key data of the cluster
	KeyData string `gorm:"type:text" json:"keyData,omitempty"`
	// serviceaccount token to connect with agent clustr
	Token string `gorm:"type:text" json:"token,omitempty"`
	// the kubeconfig of the cluster
	Kubeconfig string `gorm:"type:text" json:"kubeconfig,omitempty"`
	// the state of the cluster
	State ClusterState `gorm:"size:32" json:"state,omitempty"`
	// the last time heartbeat from agent
	LastKeepAliveTime time.Time `json:"lastKeepAliveTime,omitempty"`
	// kubernetes version
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// ready node count of the kubernetes cluster
	ReadyNodeCount int `json:"readyNodeCount,omitempty"`
	// unhealth node count of the kubernetes cluster
	UnhealthNodeCount int `json:"unhealthNodeCount,omitempty"`
	// the allocatable cpu resources of the cluster
	AllocatableCPU string `json:"allocatableCPU,omitempty"`
	// the allocatable mem resources of the cluster
	AllocatableMemory string `json:"allocatableMemory,omitempty"`
	// the capacity resources of the cluster
	CapacityCPU string `json:"capacityCPU,omitempty"`
	// the capacity resources of the cluster
	CapacityMemory string `json:"capacityMemory,omitempty"`
	// Agent is the agent of the cluster, if use agent to connect, this field is true
	Agent bool `gorm:"default:false" json:"agent,omitempty"`
	// the agent whether to schedule the application
	Schedule bool `gorm:"default:true" json:"schedule,omitempty"`
}

func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	c.State = Pending
	return c.encodingSecureData()
}

func (c *Cluster) BeforeUpdate(tx *gorm.DB) error {
	if err := c.validate(); err != nil {
		return err
	}
	return c.encodingSecureData()
}

func (c *Cluster) AfterFind(tx *gorm.DB) error {
	return c.decodeSecureData()
}

func (c *Cluster) validate() error {
	if c.State != Pending && c.State != Approve && c.State != Reject && c.State != Offline {
		return fmt.Errorf("invalid cluster state '%s'", c.State)
	}
	return nil
}

func (c *Cluster) encodingSecureData() error {
	c.CAData = base64.StdEncoding.EncodeToString([]byte(c.CAData))
	c.CertData = base64.StdEncoding.EncodeToString([]byte(c.CertData))
	c.KeyData = base64.StdEncoding.EncodeToString([]byte(c.KeyData))
	c.Token = base64.StdEncoding.EncodeToString([]byte(c.Token))
	c.Kubeconfig = base64.StdEncoding.EncodeToString([]byte(c.Kubeconfig))
	return nil
}

func (c *Cluster) decodeSecureData() error {
	var err error
	c.CAData, err = decodeBase64(c.CAData)
	if err != nil {
		return err
	}

	c.CertData, err = decodeBase64(c.CertData)
	if err != nil {
		return err
	}

	c.KeyData, err = decodeBase64(c.KeyData)
	if err != nil {
		return err
	}

	c.Token, err = decodeBase64(c.Token)
	if err != nil {
		return err
	}

	c.Kubeconfig, err = decodeBase64(c.Kubeconfig)
	if err != nil {
		return err
	}
	return nil
}

func decodeBase64(data string) (string, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	return string(decodedData), nil
}
