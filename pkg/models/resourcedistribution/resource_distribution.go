package resourcedistribution

import (
	"time"

	"github.com/hex-techs/rocket/pkg/models"
)

// ResourceDistribution defines the desired state of Distribution.
type ResourceDistribution struct {
	models.Base
	// whether to enable the resource distribution
	Deploy bool `json:"deploy,omitempty"`
	// Resource must be the complete yaml that users want to distribute.
	Resource string `gorm:"type:longtext" json:"resource,omitempty"`
	// IncludeClusters defines the clusters that users want to distribute to.
	IncludeClusters string `json:"targets,omitempty"`
	// ExcludeClusters defines the clusters that users not want to distribute to.
	ExcludeClusters string `json:"excludeClusters,omitempty"`
	// it is used to record the last time when the application is deployed
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
	// it is used to record the error message when the application is failed to deploy
	StatusMessage string `json:"message,omitempty"`
}
