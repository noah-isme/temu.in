package models

import (
    "time"

    "gorm.io/gorm"
)

type AdminAudit struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    ActorID   uint           `json:"actor_id"`
    Action    string         `json:"action"`
    Target    string         `json:"target"` // e.g., user:email
    Details   string         `json:"details"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
