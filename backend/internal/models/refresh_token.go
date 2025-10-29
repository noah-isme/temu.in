package models

import (
    "time"

    "gorm.io/gorm"
)

type RefreshToken struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    TokenHash string    `gorm:"index;not null" json:"-"`
    UserID    uint      `gorm:"index;not null" json:"user_id"`
    ExpiresAt time.Time `json:"expires_at"`
    Revoked   bool      `gorm:"default:false" json:"revoked"`
}
