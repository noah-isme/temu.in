package models

import "time"

const (
	NotificationChannelEmail = "email"

	NotificationStatusPending    = "pending"
	NotificationStatusProcessing = "processing"
	NotificationStatusRetry      = "retry"
	NotificationStatusSent       = "sent"
	NotificationStatusDeadLetter = "dead_letter"

	NotificationDeliverySent       = "sent"
	NotificationDeliveryRetry      = "retry"
	NotificationDeliveryDeadLetter = "dead_letter"
)

const (
	NotificationTypeBookingConfirmationCustomer = "booking_confirmation_customer"
	NotificationTypeBookingConfirmationProvider = "booking_confirmation_provider"
	NotificationTypeBookingCancellationCustomer = "booking_cancellation_customer"
	NotificationTypeBookingCancellationProvider = "booking_cancellation_provider"
	NotificationTypeBookingReminderCustomer     = "booking_reminder_customer"
	NotificationTypeBookingReminderProvider     = "booking_reminder_provider"
	NotificationTypePaymentReceipt              = "payment_receipt"
	NotificationTypeEmailVerification           = "email_verification"
	NotificationTypePasswordReset               = "password_reset"
)

// Notification is the durable outbox record. A notification is not sent by
// the HTTP request; it is claimed and delivered by the worker after commit.
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Type              string     `gorm:"type:varchar(80);not null;index" json:"type"`
	Channel           string     `gorm:"type:varchar(20);not null;default:'email'" json:"channel"`
	DeduplicationKey  string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	RecipientEmail    string     `gorm:"type:varchar(255);not null" json:"recipient_email"`
	RecipientName     string     `gorm:"type:varchar(255)" json:"recipient_name"`
	Subject           string     `gorm:"type:varchar(255);not null" json:"subject"`
	Template          string     `gorm:"type:varchar(120);not null" json:"template"`
	Payload           string     `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	Status            string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Attempts          int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt     *time.Time `gorm:"index" json:"next_attempt_at"`
	LockedAt          *time.Time `json:"-"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	LastError         string     `gorm:"type:text" json:"last_error,omitempty"`
	ProviderMessageID string     `gorm:"type:varchar(255)" json:"provider_message_id,omitempty"`

	BookingID  *uint   `gorm:"index" json:"booking_id,omitempty"`
	UserID     *uint   `gorm:"index" json:"user_id,omitempty"`
	ProviderID *string `gorm:"type:uuid;index" json:"provider_id,omitempty"`
}

// NotificationDelivery records the result of every delivery attempt,
// including retries and dead-letter transitions.
type NotificationDelivery struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	NotificationID    uint      `gorm:"not null;index" json:"notification_id"`
	Attempt           int       `gorm:"not null" json:"attempt"`
	Status            string    `gorm:"type:varchar(20);not null" json:"status"`
	ProviderMessageID string    `gorm:"type:varchar(255)" json:"provider_message_id,omitempty"`
	Error             string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

func (NotificationDelivery) TableName() string { return "notification_deliveries" }
