package publicresource

import (
	"errors"

	"github.com/go-faker/faker/v4"
	"github.com/hex-techs/rocket/pkg/models"
	"gorm.io/gorm"
)

type ResourceType string

const (
	// the resource is a region
	Region ResourceType = "Region"
)

type PublicResource struct {
	models.Base
	Type        ResourceType `gorm:"index" json:"type,omitempty"`
	Name        string       `gorm:"index" json:"name,omitempty"`
	DisplayName string       `json:"displayName,omitempty"`
	Description string       `json:"description,omitempty"`
}

func (pr *PublicResource) BeforeCreate(tx *gorm.DB) (err error) {
	var existingResource PublicResource
	if err := tx.Where("type = ? AND name = ?", pr.Type, pr.Name).First(&existingResource).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		return errors.New("resource with same type and name already exists")
	}
	return nil
}

// FakeData generate fake data
func FakeData(count int) []*PublicResource {
	var res []*PublicResource
	for i := 0; i < count; i++ {
		name := faker.GetRealAddress().City
		res = append(res, &PublicResource{
			Type:        Region,
			Name:        name,
			DisplayName: name,
			Description: faker.LastName(),
		})
	}
	return res
}
