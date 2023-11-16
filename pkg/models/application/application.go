package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hex-techs/rocket/pkg/models"
	"github.com/hex-techs/rocket/pkg/models/project"
	"gorm.io/gorm"
)

type ApplicationEnvironment string

const (
	Dev     ApplicationEnvironment = "dev"
	Test    ApplicationEnvironment = "test"
	Staging ApplicationEnvironment = "staging"
	Prod    ApplicationEnvironment = "prod"
)

type ApplicationPhase string

const (
	// the application is wait for schedule
	Pending ApplicationPhase = "Pending"
	// the application is in scheduling
	Scheduling ApplicationPhase = "Scheduling"
	// the application is scheduled
	Scheduled ApplicationPhase = "Scheduled"
	// the application is Descheduling
	Descheduling ApplicationPhase = "Descheduling"
)

// Application defines the desired state of Application
type Application struct {
	models.Base
	models.Finalizer
	// Name is the name of the application
	Name string `gorm:"size:128,not null" json:"name,omitempty" binding:"required"`
	// Namespace is the namespace of the application
	Namespace string `gorm:"size:128,not null" json:"namespace,omitempty" binding:"required"`
	// ProjectID is the application project belongs to
	ProjectID uint   `gorm:"size:128,not null" json:"projectId,omitempty" binding:"required"`
	Project   string `gorm:"-" json:"project,omitempty"`
	// Description is the description of the application
	Description string `gorm:"size:1024" json:"description,omitempty"`
	// Which cloud area the application in
	CloudArea string `json:"cloudArea,omitempty" binding:"required"`
	// prod, pre or test
	Environment ApplicationEnvironment `json:"environment,omitempty" binding:"required"`
	// The region of application
	RegionInfo string   `json:"-"`
	Regions    []string `gorm:"-" json:"regions,omitempty"`
	// Template must be the complete yaml that users want to distribute.
	TemplateRaw string    `gorm:"type:longtext" json:"-"`
	Template    *Workload `gorm:"-" json:"template,omitempty"`
	// the cluster of application
	Cluster string `json:"cluster,omitempty"`
	// LastSchedulerCluster is the cluster that the application last scheduled to.
	LastSchedulerClusters string `json:"lastSchedulerClusters,omitempty"`
	// the phase of the Application
	Phase ApplicationPhase `json:"phase,omitempty"`
	// application details
	ApplicationDetails string `json:"applicationDetails,omitempty"`
	// it is used to record the last time when the application is deployed
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
	// it is used to record the error message when the application is failed to deploy
	StatusMessage string `json:"message,omitempty"`

	Traits string `gorm:"type:longtext" json:"-"`

	Scale                *Scale                `gorm:"-" json:"scale,omitempty"`
	Metrics              *Metrics              `gorm:"-" json:"metrics,omitempty"`
	Tolerate             *Tolerate             `gorm:"-" json:"tolerate,omitempty"`
	Affinity             *Affinity             `gorm:"-" json:"affinity,omitempty"`
	Probe                *Probe                `gorm:"-" json:"probe,omitempty"`
	PodUnavailableBudget *PodUnavailableBudget `gorm:"-" json:"podUnavailableBudget,omitempty"`
	Service              *Service              `gorm:"-" json:"service,omitempty"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	return a.writeOption(tx)
}

func (a *Application) BeforeUpdate(tx *gorm.DB) error {
	return a.writeOption(tx)
}

func (a *Application) writeOption(tx *gorm.DB) error {
	var project project.Project
	if err := tx.Where("id = ?", a.ProjectID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project not found")
		}
		return err
	}
	var old Application
	if err := tx.Model(&a).Where("name = ? and namespace = ? and project_id = ? and cloud_area = ? and region_info = ? and cluster = ?",
		a.Name, a.Namespace, a.ProjectID, a.CloudArea, a.RegionInfo, a.Cluster).First(&old).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if old.ID != 0 && old.ID != a.ID {
		return errors.New("application already exists")
	}
	a.RegionInfo = strings.Join(a.Regions, ",")
	b, err := json.Marshal(a.Template)
	if err != nil {
		return fmt.Errorf("marshal resource error: %w", err)
	}
	a.TemplateRaw = string(b)
	a.convertTrait2Raw()
	return nil
}

func (a *Application) AfterFind(tx *gorm.DB) error {
	var project project.Project
	if err := tx.Where("id = ?", a.ProjectID).First(&project).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	a.Project = project.Name
	if a.Project == "" {
		a.Project = "Unknown"
	}
	a.Regions = strings.Split(a.RegionInfo, ",")
	// b, err := json.Marshal(a.TemplateRaw)
	// if err != nil {
	// 	return fmt.Errorf("marshal template error: %w", err)
	// }
	if err := json.Unmarshal([]byte(a.TemplateRaw), &a.Template); err != nil {
		return fmt.Errorf("unmarshal template error: %w", err)
	}
	return a.convertTrait2Json()
}

func (a *Application) convertTrait2Json() error {
	var traits []TraitTemplate
	if err := json.Unmarshal([]byte(a.Traits), &traits); err != nil {
		return fmt.Errorf("unmarshal traits error: %w", err)
	}
	for i := range traits {
		if err := a.handleTrait(&traits[i]); err != nil {
			return err
		}
	}
	return nil
}
func (a *Application) convertTrait2Raw() {
	var traits []TraitTemplate
	if a.Scale != nil {
		b, _ := json.Marshal(a.Scale)
		traits = append(traits, TraitTemplate{
			Type:     Local,
			Kind:     ScaleKind,
			Template: string(b),
		})
	}
	if a.Affinity != nil {
		b, _ := json.Marshal(a.Affinity)
		traits = append(traits, TraitTemplate{
			Type:     Local,
			Kind:     AffinityKind,
			Template: string(b),
		})
	}
	if a.Tolerate != nil {
		b, _ := json.Marshal(a.Tolerate)
		traits = append(traits, TraitTemplate{
			Type:     Local,
			Kind:     TolerateKind,
			Template: string(b),
		})
	}
	if a.Metrics != nil {
		b, _ := json.Marshal(a.Metrics)
		traits = append(traits, TraitTemplate{
			Type:     Local,
			Kind:     MetricsKind,
			Template: string(b),
		})
	}
	if a.Probe != nil {
		b, _ := json.Marshal(a.Probe)
		traits = append(traits, TraitTemplate{
			Type:     Local,
			Kind:     ProbeKind,
			Template: string(b),
		})
	}
	if a.PodUnavailableBudget != nil {
		b, _ := json.Marshal(a.PodUnavailableBudget)
		traits = append(traits, TraitTemplate{
			Type:     Edge,
			Kind:     PodBudgetKind,
			Template: string(b),
		})
	}
	if a.Service != nil {
		b, _ := json.Marshal(a.Service)
		traits = append(traits, TraitTemplate{
			Type:     Edge,
			Kind:     ServiceKind,
			Template: string(b),
		})
	}
	b, _ := json.Marshal(traits)
	a.Traits = string(b)
}

func (a *Application) handleTrait(trait *TraitTemplate) error {
	switch trait.Kind {
	case ScaleKind:
		var scale Scale
		if err := json.Unmarshal([]byte(trait.Template), &scale); err != nil {
			return fmt.Errorf("unmarshal scale error: %w", err)
		}
		a.Scale = &scale
	case AffinityKind:
		var affinity Affinity
		if err := json.Unmarshal([]byte(trait.Template), &affinity); err != nil {
			return fmt.Errorf("unmarshal affinity error: %w", err)
		}
		a.Affinity = &affinity
	case TolerateKind:
		var tolerate Tolerate
		if err := json.Unmarshal([]byte(trait.Template), &tolerate); err != nil {
			return fmt.Errorf("unmarshal tolerate error: %w", err)
		}
		a.Tolerate = &tolerate
	case MetricsKind:
		var metrics Metrics
		if err := json.Unmarshal([]byte(trait.Template), &metrics); err != nil {
			return fmt.Errorf("unmarshal metrics error: %w", err)
		}
		a.Metrics = &metrics
	case ProbeKind:
		var probe Probe
		if err := json.Unmarshal([]byte(trait.Template), &probe); err != nil {
			return fmt.Errorf("unmarshal probe error: %w", err)
		}
		a.Probe = &probe
	case PodBudgetKind:
		var podUnavailableBudget PodUnavailableBudget
		if err := json.Unmarshal([]byte(trait.Template), &podUnavailableBudget); err != nil {
			return fmt.Errorf("unmarshal podUnavailableBudget error: %w", err)
		}
		a.PodUnavailableBudget = &podUnavailableBudget
	case ServiceKind:
		var service Service
		if err := json.Unmarshal([]byte(trait.Template), &service); err != nil {
			return fmt.Errorf("unmarshal service error: %w", err)
		}
		a.Service = &service
	}
	return nil
}

func (a *Application) GetName() string {
	return a.Name
}

func (a *Application) GetNamespace() string {
	return a.Namespace
}
