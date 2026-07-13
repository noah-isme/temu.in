package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	ID          string          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProviderID  string          `json:"provider_id" gorm:"type:uuid;not null"`
	Provider    ServiceProvider `json:"provider" gorm:"foreignKey:ProviderID"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	Description string          `json:"description" gorm:"type:text"`
	Duration    int             `json:"duration" gorm:"not null"` // minutes
	Price       float64         `json:"price" gorm:"type:decimal(10,2);not null"`
	IsActive    bool            `json:"is_active" gorm:"default:true"`
	Category    string          `json:"category,omitempty" gorm:"type:varchar(100)"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Service) TableName() string {
	return "services"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (s *Service) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return
}
