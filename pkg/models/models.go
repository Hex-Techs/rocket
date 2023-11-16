package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Base
type Base struct {
	// default primary key
	ID        uint           `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt time.Time      `json:"createdAt,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Finalizer
type Finalizer struct {
	Finalizers        []string `gorm:"-" json:"finalizers"`
	InternalFinalizer string   `json:"-"`
}

// AfterFind
func (f *Finalizer) AfterFind(tx *gorm.DB) error {
	return f.convertToExternal(tx)
}

func (f *Finalizer) convertToInternal(tx *gorm.DB) error {
	f.InternalFinalizer = strings.Join(f.Finalizers, ",")
	return nil
}

func (f *Finalizer) convertToExternal(tx *gorm.DB) error {
	f.Finalizers = strings.Split(f.InternalFinalizer, ",")
	return nil
}
