package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"temu.in/backend/internal/models"
)

type CreateRequest struct {
	UserID      uint
	ProviderID  string
	ServiceID   string
	Amount      int64
	Currency    string
	Description string
}

type Service struct {
	repo     *Repository
	provider Provider
}

func NewService(repo *Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

// Prepare creates a provider transaction but does not persist local state. The
// booking repository persists the returned payment and booking together, so a
// payment can never be linked to a partially-created booking.
func (s *Service) Prepare(ctx context.Context, req CreateRequest) (*models.Payment, error) {
	if s.provider == nil {
		return nil, ErrProviderNotConfigured
	}
	if req.UserID == 0 || req.Amount <= 0 || req.Currency == "" {
		return nil, errors.New("invalid payment request")
	}

	orderID, err := NewOrderID()
	if err != nil {
		return nil, fmt.Errorf("generate payment order id: %w", err)
	}
	transaction, err := s.provider.CreateTransaction(ctx, CreateTransactionRequest{
		OrderID:     orderID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, errors.New("payment provider returned an empty transaction")
	}
	status := transaction.Status
	if status == "" {
		status = models.PaymentStatusPending
	}
	return &models.Payment{
		UserID:        req.UserID,
		Provider:      ProviderMidtrans,
		OrderID:       orderID,
		IntentID:      orderID,
		ClientSecret:  transaction.Token,
		RedirectURL:   transaction.RedirectURL,
		TransactionID: transaction.TransactionID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        status,
	}, nil
}

func (s *Service) GetTransaction(ctx context.Context, orderID string) (*Transaction, error) {
	if s.provider == nil {
		return nil, ErrProviderNotConfigured
	}
	return s.provider.GetTransaction(ctx, orderID)
}

func (s *Service) VerifyWebhook(notification WebhookNotification) bool {
	return s.provider != nil && s.provider.VerifyWebhook(notification)
}

func NewOrderID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "temu_" + hex.EncodeToString(random[:]), nil
}
