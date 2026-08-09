package payment

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"temu.in/backend/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(p *models.Payment) error {
	return r.db.Create(p).Error
}

func (r *Repository) Update(p *models.Payment) error {
	return r.db.Save(p).Error
}

func (r *Repository) FindByIntent(intent string) (*models.Payment, error) {
	var p models.Payment
	if err := r.db.Where("intent_id = ?", intent).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) FindByClientSecret(cs string) (*models.Payment, error) {
	var p models.Payment
	if err := r.db.Where("client_secret = ?", cs).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) FindByOrderID(orderID string) (*models.Payment, error) {
	var p models.Payment
	if err := r.db.Where("order_id = ?", orderID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) FindByID(id uint) (*models.Payment, error) {
	var p models.Payment
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProviderStatus is idempotent and never moves a payment backwards in
// its state machine. Replayed or out-of-order provider notifications are safe.
func (r *Repository) UpdateProviderStatus(id uint, status, transactionID string) (*models.Payment, error) {
	var p models.Payment
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error; err != nil {
			return err
		}
		if !models.CanTransitionPaymentStatus(p.Status, status) {
			return nil
		}
		updates := map[string]interface{}{"status": status}
		if transactionID != "" {
			updates["transaction_id"] = transactionID
		}
		if err := tx.Model(&models.Payment{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		p.Status = status
		if transactionID != "" {
			p.TransactionID = transactionID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// RecordWebhookEvent atomically claims an event key. A duplicate key means
// the notification has already been handled and is not an error.
func (r *Repository) RecordWebhookEvent(eventKey string, paymentID uint) (bool, error) {
	event := &models.PaymentWebhookEvent{EventKey: eventKey, PaymentID: paymentID}
	if err := r.db.Create(event).Error; err != nil {
		var stateErr interface{ SQLState() string }
		if errors.As(err, &stateErr) && stateErr.SQLState() == "23505" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DB exposes underlying gorm DB for limited internal use
func (r *Repository) DB() *gorm.DB { return r.db }
