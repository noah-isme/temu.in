package models

import (
	"time"

	"github.com/google/uuid"
)

type Favorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uint      `gorm:"not null" json:"userId"`
	ServiceID *string   `json:"serviceId,omitempty"` // Nullable if favoring provider

	ProviderID *string   `json:"providerId,omitempty"` // Nullable if favoring service
	CreatedAt  time.Time `json:"createdAt"`

	// Associations
	User     User             `gorm:"foreignKey:UserID" json:"-"`
	Service  *Service         `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	Provider *ServiceProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
