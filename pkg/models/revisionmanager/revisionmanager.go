package revisionmanager

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	defaultRevisionManagerLimit = 10
)

// RevisionManager is the Schema for the revisionmanagers to store the revisions
type RevisionManager struct {
	ID           uint      `gorm:"primary_key" json:"id,omitempty"`
	ResourceKind string    `gorm:"size:128" json:"resourceKind,omitempty" binding:"required"`
	Cluster      string    `gorm:"size:128" json:"cluster,omitempty"`
	Namespace    string    `json:"namespace,omitempty"`
	ResourceName string    `gorm:"size:128" json:"resourceName,omitempty" binding:"required"`
	ResourceID   uint      `json:"resourceId,omitempty"`
	Version      string    `gorm:"size:128" json:"version,omitempty" binding:"required"`
	Content      string    `gorm:"type:text" json:"content,omitempty" binding:"required"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
}

// BefaoreCreate
func (r *RevisionManager) BeforeCreate(tx *gorm.DB) error {
	var count int64
	condition := fmt.Sprintf("resource_kind = '%s' and resource_name = '%s' and cluster = '%s' and namespace = '%s'",
		r.ResourceKind, r.ResourceName, r.Cluster, r.Namespace)
	if err := tx.Model(&RevisionManager{}).Where(condition).Count(&count); err != nil {
		return fmt.Errorf("failed to validate revisions when create: %v", err)
	}
	if count >= defaultRevisionManagerLimit {
		// delete the oldest revision
		var rm RevisionManager
		if err := tx.Model(&RevisionManager{}).Order("create_at asc").Where(condition).First(&rm).Error; err != nil {
			return fmt.Errorf("failed to find revisions when create: %v", err)
		}
		if err := tx.Delete(&rm).Error; err != nil {
			return fmt.Errorf("failed to delete revisions when create: %v", err)
		}
	}
	r.CreatedAt = time.Now()
	return nil
}

// BefaoreUpdate
func (r *RevisionManager) BeforeUpdate(tx *gorm.DB) error {
	return errors.New("RevisionManager is immutable")
}
