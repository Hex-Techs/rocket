package project

import (
	"fmt"
	"strings"

	"github.com/hex-techs/rocket/pkg/models"
	"github.com/hex-techs/rocket/pkg/models/module"
	resouce "github.com/hex-techs/rocket/pkg/utils/resource"
	"gorm.io/gorm"
)

type Project struct {
	models.Base
	Name              string `gorm:"unique,index,size:64,not null" json:"name,omitempty" binding:"required"`
	DisplayName       string `gorm:"size:64;not null;unique" json:"displayName,omitempty" binding:"required"`
	Description       string `gorm:"size:1024" json:"description,omitempty"`
	CPUUsage          string `gorm:"-" json:"cpuUsage,omitempty"`
	MemoryUsage       string `gorm:"-" json:"memoryUsage,omitempty"`
	CPURequirement    string `gorm:"size:32" json:"cpuRequirement,omitempty"`
	MemoryRequirement string `gorm:"size:32" json:"memoryRequirement,omitempty"`
	// 开发语言
	Language string `gorm:"size:32" json:"language,omitempty"`
	// 开发框架
	Framework string `gorm:"size:32" json:"framework,omitempty"`
	// 负责人
	Owner  string   `gorm:"size:1024" json:"-"`
	Owners []string `gorm:"-" json:"owners,omitempty"`
	// 产品负责人
	ProductOwner string   `gorm:"size:1024" json:"-"`
	Producters   []string `gorm:"-" json:"producters,omitempty"`
	// 测试负责人
	TestOwner string   `gorm:"size:1024" json:"-"`
	Testers   []string `gorm:"-" json:"testers,omitempty"`
	// SRE
	SRE  string   `gorm:"size:1024" json:"-"`
	SREs []string `gorm:"-" json:"sres,omitempty"`
	// 所属模块 id
	ModuleID uint `gorm:"size:256;not null" json:"moduleID,omitempty"`
	// 所属模块
	Module string `gorm:"-" json:"module,omitempty"`
	// 标签组
	Tags string `gorm:"size:1024" json:"tags,omitempty"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	return p.handleWrite()
}

func (p *Project) AfterFind(tx *gorm.DB) error {
	var md module.Module
	r := tx.Model(md).Where("id = ?", p.ModuleID).First(&md)
	if r.Error != nil {
		return r.Error
	}
	p.Module = md.FullName
	p.Owners = strings.Split(p.Owner, ",")
	p.Producters = strings.Split(p.ProductOwner, ",")
	p.Testers = strings.Split(p.TestOwner, ",")
	p.SREs = strings.Split(p.SRE, ",")
	return nil
}

func (p *Project) BeforeUpdate(tx *gorm.DB) error {
	return p.handleWrite()
}

func (p *Project) handleWrite() error {
	if err := p.validate(); err != nil {
		return err
	}
	p.Owners = strings.Split(p.Owner, ",")
	p.Producters = strings.Split(p.ProductOwner, ",")
	p.Testers = strings.Split(p.TestOwner, ",")
	p.SREs = strings.Split(p.SRE, ",")
	return nil
}

func (p *Project) validate() error {
	_, err := resouce.ResourceEngine.GetVar(p.CPURequirement)
	if err != nil {
		return fmt.Errorf("cpuRequirement: %s", err.Error())
	}
	_, err = resouce.ResourceEngine.GetVar(p.MemoryRequirement)
	if err != nil {
		return fmt.Errorf("memoryRequirement: %s", err.Error())
	}
	return nil
}
