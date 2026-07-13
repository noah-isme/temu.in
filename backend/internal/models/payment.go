package models

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	IntentID     string `json:"intent_id"` // internal id
	ClientSecret string `json:"client_secret"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	Status       string `json:"status"` // created, succeeded, failed
}
