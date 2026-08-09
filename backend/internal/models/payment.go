package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	PaymentStatusCreated   = "created"
	PaymentStatusPending   = "pending"
	PaymentStatusSucceeded = "succeeded"
	PaymentStatusFailed    = "failed"
	PaymentStatusExpired   = "expired"
	PaymentStatusCancelled = "cancelled"
	PaymentStatusRefunded  = "refunded"
)

type Payment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID        uint   `gorm:"index" json:"-"`
	Provider      string `json:"provider"`
	OrderID       string `gorm:"uniqueIndex" json:"order_id"`
	IntentID      string `json:"intent_id"` // legacy alias for OrderID
	ClientSecret  string `json:"client_secret"`
	RedirectURL   string `json:"redirect_url,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
}

// PaymentWebhookEvent records a verified provider notification. Midtrans does
// not expose a separate event id, so the webhook handler stores a digest of the
// signed payload as the idempotency key.
type PaymentWebhookEvent struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EventKey  string    `gorm:"uniqueIndex;not null" json:"-"`
	PaymentID uint      `gorm:"not null;index" json:"payment_id"`
	CreatedAt time.Time `json:"created_at"`
}

func CanTransitionPaymentStatus(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case PaymentStatusCreated:
		return to == PaymentStatusPending || to == PaymentStatusSucceeded || to == PaymentStatusFailed || to == PaymentStatusExpired || to == PaymentStatusCancelled
	case PaymentStatusPending:
		return to == PaymentStatusSucceeded || to == PaymentStatusFailed || to == PaymentStatusExpired || to == PaymentStatusCancelled
	case PaymentStatusSucceeded:
		return to == PaymentStatusRefunded
	case PaymentStatusFailed, PaymentStatusExpired, PaymentStatusCancelled, PaymentStatusRefunded:
		return false
	default:
		return false
	}
}
