package module

import (
	"fmt"
	"time"

	"github.com/hex-techs/rocket/pkg/models"
	"gorm.io/gorm"
)

type Module struct {
	models.Base
	// 名称
	Name        string `gorm:"size:128;not null;unique;index" json:"name,omitempty" binding:"required"`
	DisplayName string `gorm:"size:128;not null;unique" json:"displayName,omitempty" binding:"required"`
	// 描述
	Description string `gorm:"size:1024" json:"description,omitempty"`
	// 父模块id
	ParentID uint `gorm:"index" json:"parentID,omitempty"`
	// 级别
	Level uint `gorm:"index" json:"level,omitempty"`
	// module全称
	FullName string `json:"fullName,omitempty"`
}

func (m *Module) BeforeCreate(tx *gorm.DB) error {
	if m.ParentID == 0 {
		m.Level = 1
		m.CreatedAt = time.Now()
		return nil
	}
	var parent Module
	r := tx.Model(m).Where("id = ?", m.ParentID).First(&parent)
	if r.Error != nil {
		return fmt.Errorf("parent module error: %v", r.Error)
	}
	m.Level = parent.Level + 1
	if m.Level > 5 {
		// 最多支持5级
		return fmt.Errorf("module level more than 5")
	}
	return nil
}

func (m *Module) AfterFind(tx *gorm.DB) error {
	t, err := m.findParent(tx)
	if err != nil {
		return err
	}
	m.FullName = fmt.Sprintf("%s/%s", t, m.Name)
	return nil
}

func (m *Module) findParent(tx *gorm.DB) (string, error) {
	if m.ParentID == 0 {
		return "", nil
	}
	var parent Module
	r := tx.Model(m).Where("id = ?", m.ParentID).First(&parent)
	if r.Error != nil {
		return "", fmt.Errorf("parent module error: %v", r.Error)
	}
	if parent.ParentID == 0 {
		return parent.Name, nil
	} else {
		n, err := parent.findParent(tx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s/%s", n, parent.Name), nil
	}
}
