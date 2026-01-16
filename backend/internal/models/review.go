package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Review struct {
	ID         string          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  gorm.DeletedAt  `json:"-" gorm:"index"`
	BookingID  uint            `json:"booking_id" gorm:"not null;unique"`
	Booking    Booking         `json:"booking" gorm:"foreignKey:BookingID"`
	ProviderID string          `json:"provider_id" gorm:"type:uuid;not null"`
	Provider   ServiceProvider `json:"provider" gorm:"foreignKey:ProviderID"`
	UserID     uint            `json:"user_id" gorm:"not null"`
	User       User            `json:"user" gorm:"foreignKey:UserID"`
	Rating     int             `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment    string          `json:"comment" gorm:"type:text"`
}

// TableName specifies the table name for GORM
func (Review) TableName() string {
	return "reviews"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (r *Review) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}
