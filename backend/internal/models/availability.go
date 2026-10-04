package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AvailabilitySchedule struct {
	ID          string          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProviderID  string          `json:"provider_id" gorm:"type:uuid;not null"`
	Provider    ServiceProvider `json:"provider" gorm:"foreignKey:ProviderID"`
	DayOfWeek   int             `json:"day_of_week" gorm:"not null"`          // 0=Sunday, 1=Monday, ..., 6=Saturday
	StartTime   string          `json:"start_time" gorm:"type:time;not null"` // HH:MM format
	EndTime     string          `json:"end_time" gorm:"type:time;not null"`   // HH:MM format
	IsAvailable bool            `json:"is_available"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (AvailabilitySchedule) TableName() string {
	return "availability_schedules"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (a *AvailabilitySchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return
}

type AvailabilityException struct {
	ID          string          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProviderID  string          `json:"provider_id" gorm:"type:uuid;not null"`
	Provider    ServiceProvider `json:"provider" gorm:"foreignKey:ProviderID"`
	Date        time.Time       `json:"date" gorm:"type:date;not null"`
	StartTime   string          `json:"start_time" gorm:"type:time"` // nullable for full day
	EndTime     string          `json:"end_time" gorm:"type:time"`   // nullable for full day
	IsAvailable bool            `json:"is_available" gorm:"default:false"`
	Reason      string          `json:"reason" gorm:"type:text"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (AvailabilityException) TableName() string {
	return "availability_exceptions"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (a *AvailabilityException) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return
}
