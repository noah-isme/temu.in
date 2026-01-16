package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceProvider struct {
	ID           string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       string    `json:"user_id" gorm:"type:uuid;not null"`
	User         User      `json:"user" gorm:"foreignKey:UserID"`
	BusinessName string    `json:"business_name" gorm:"type:varchar(255);not null"`
	BusinessType string    `json:"business_type" gorm:"type:varchar(50);not null"` // doctor, salon, consultant, studio
	Description  string    `json:"description" gorm:"type:text"`
	Address      string    `json:"address" gorm:"type:text"`
	Latitude     *float64  `json:"latitude" gorm:"type:decimal(10,8)"`
	Longitude    *float64  `json:"longitude" gorm:"type:decimal(11,8)"`
	PricingInfo  string    `json:"pricing_info" gorm:"type:jsonb"` // JSON string for pricing details
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (ServiceProvider) TableName() string {
	return "service_providers"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (sp *ServiceProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if sp.ID == "" {
		sp.ID = uuid.New().String()
	}
	return
}
