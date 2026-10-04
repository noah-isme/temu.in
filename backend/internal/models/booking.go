package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	BookingStatusPending       = "pending"
	BookingStatusConfirmed     = "confirmed"
	BookingStatusCompleted     = "completed"
	BookingStatusCancelled     = "cancelled"
	BookingStatusPaymentFailed = "payment_failed"

	BookingPaymentUnpaid      = "unpaid"
	BookingPaymentPending     = "pending"
	BookingPaymentPaid        = "paid"
	BookingPaymentFailed      = "failed"
	BookingPaymentNotRequired = "not_required"
)

type Booking struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID          uint            `gorm:"not null;index:idx_bookings_customer" json:"user_id"`
	User            User            `json:"user" gorm:"foreignKey:UserID"`
	ProviderID      string          `gorm:"not null;index:idx_bookings_provider_date" json:"provider_id"`
	ServiceProvider ServiceProvider `json:"service_provider" gorm:"foreignKey:ProviderID"`
	ServiceID       *string         `json:"service_id,omitempty"`
	// Service is preloaded only by the read-only "*WithNames" finders that
	// build response DTOs; write paths never populate it.
	Service *Service `json:"-" gorm:"foreignKey:ServiceID"`

	CustomerName string    `json:"customer_name"`
	BookingDate  time.Time `gorm:"type:date;index:idx_bookings_provider_date" json:"booking_date"`
	StartTime    time.Time `gorm:"type:time;serializer:clock" json:"start_time"`
	EndTime      time.Time `gorm:"type:time;serializer:clock" json:"end_time"`
	Slot         string    `json:"slot,omitempty"`

	Status             string `gorm:"type:varchar(20);default:'pending';index:idx_bookings_status" json:"status"`
	CustomerNotes      string `gorm:"type:text" json:"customer_notes,omitempty"`
	AdminNotes         string `gorm:"type:text" json:"admin_notes,omitempty"`
	CancellationReason string `gorm:"type:text" json:"cancellation_reason,omitempty"`

	PaymentStatus string `gorm:"type:varchar(20);default:'unpaid'" json:"payment_status"`
	PaymentID     *uint  `json:"payment_id,omitempty"`
}

func CanTransitionBookingStatus(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case BookingStatusPending:
		return to == BookingStatusConfirmed || to == BookingStatusCancelled || to == BookingStatusPaymentFailed
	case BookingStatusConfirmed:
		return to == BookingStatusCompleted || to == BookingStatusCancelled
	case BookingStatusPaymentFailed:
		return to == BookingStatusCancelled
	case BookingStatusCompleted, BookingStatusCancelled:
		return false
	default:
		return false
	}
}

type ProviderStats struct {
	TotalBookings     int64 `json:"total_bookings"`
	PendingBookings   int64 `json:"pending_bookings"`
	CompletedBookings int64 `json:"completed_bookings"`
	TotalRevenue      int64 `json:"total_revenue"`
}
