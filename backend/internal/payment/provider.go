package payment

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrProviderNotConfigured = errors.New("payment provider is not configured")

const ProviderMidtrans = "midtrans"

type CreateTransactionRequest struct {
	OrderID     string
	Amount      int64
	Currency    string
	Description string
}

type Transaction struct {
	OrderID       string
	TransactionID string
	Token         string
	RedirectURL   string
	Status        string
}

type WebhookNotification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	SignatureKey      string `json:"signature_key"`
}

// Provider isolates payment gateway behavior from local payment and booking
// state. A test provider can implement this interface without changing the
// booking flow; production uses MidtransProvider below.
type Provider interface {
	CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*Transaction, error)
	GetTransaction(ctx context.Context, orderID string) (*Transaction, error)
	VerifyWebhook(notification WebhookNotification) bool
}

type MidtransProvider struct {
	serverKey string
	clientKey string
	baseURL   string
	client    *http.Client
}

func NewMidtransProvider(serverKey, clientKey, baseURL string) *MidtransProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.sandbox.midtrans.com"
	}
	return &MidtransProvider{
		serverKey: serverKey,
		clientKey: clientKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *MidtransProvider) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*Transaction, error) {
	if strings.TrimSpace(p.serverKey) == "" {
		return nil, ErrProviderNotConfigured
	}
	if req.OrderID == "" || req.Amount <= 0 {
		return nil, errors.New("invalid payment transaction")
	}
	if req.Currency != "" && !strings.EqualFold(req.Currency, "IDR") {
		return nil, errors.New("midtrans supports IDR payments only")
	}

	body := struct {
		TransactionDetails struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		} `json:"transaction_details"`
		ItemDetails []struct {
			ID       string `json:"id"`
			Price    int64  `json:"price"`
			Quantity int    `json:"quantity"`
			Name     string `json:"name"`
		} `json:"item_details,omitempty"`
	}{
		TransactionDetails: struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		}{OrderID: req.OrderID, GrossAmount: req.Amount},
	}
	if req.Description != "" {
		body.ItemDetails = []struct {
			ID       string `json:"id"`
			Price    int64  `json:"price"`
			Quantity int    `json:"quantity"`
			Name     string `json:"name"`
		}{{ID: req.OrderID, Price: req.Amount, Quantity: 1, Name: req.Description}}
	}

	var response struct {
		Token       string `json:"token"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := p.doJSON(ctx, http.MethodPost, "/snap/v1/transactions", body, &response); err != nil {
		return nil, err
	}
	if response.Token == "" && response.RedirectURL == "" {
		return nil, errors.New("midtrans returned no payment token")
	}
	return &Transaction{OrderID: req.OrderID, Token: response.Token, RedirectURL: response.RedirectURL, Status: "pending"}, nil
}

func (p *MidtransProvider) GetTransaction(ctx context.Context, orderID string) (*Transaction, error) {
	if strings.TrimSpace(p.serverKey) == "" {
		return nil, ErrProviderNotConfigured
	}
	if orderID == "" {
		return nil, errors.New("order id is required")
	}

	var response struct {
		OrderID           string `json:"order_id"`
		TransactionID     string `json:"transaction_id"`
		TransactionStatus string `json:"transaction_status"`
		FraudStatus       string `json:"fraud_status"`
	}
	if err := p.doJSON(ctx, http.MethodGet, "/v2/"+url.PathEscape(orderID)+"/status", nil, &response); err != nil {
		return nil, err
	}
	return &Transaction{
		OrderID:       response.OrderID,
		TransactionID: response.TransactionID,
		Status:        normalizeProviderStatus(response.TransactionStatus, response.FraudStatus),
	}, nil
}

func (p *MidtransProvider) VerifyWebhook(notification WebhookNotification) bool {
	if strings.TrimSpace(p.serverKey) == "" || notification.OrderID == "" || notification.SignatureKey == "" {
		return false
	}
	digest := sha512.Sum512([]byte(notification.OrderID + notification.StatusCode + notification.GrossAmount + p.serverKey))
	expected := hex.EncodeToString(digest[:])
	provided := strings.TrimSpace(strings.ToLower(notification.SignatureKey))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func (p *MidtransProvider) doJSON(ctx context.Context, method, path string, requestBody, responseBody interface{}) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode payment request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create payment request: %w", err)
	}
	auth := base64.StdEncoding.EncodeToString([]byte(p.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("payment provider request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("payment provider returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode payment provider response: %w", err)
	}
	return nil
}

func normalizeProviderStatus(transactionStatus, fraudStatus string) string {
	switch strings.ToLower(transactionStatus) {
	case "settlement":
		return "succeeded"
	case "capture":
		switch strings.ToLower(fraudStatus) {
		case "deny":
			return "failed"
		case "challenge":
			// Funds are held pending a manual fraud review; not paid yet.
			return "pending"
		}
		return "succeeded"
	case "pending":
		return "pending"
	case "expire":
		return "expired"
	case "cancel", "deny", "failure":
		return "failed"
	case "refund", "partial_refund", "chargeback", "partial_chargeback":
		return "refunded"
	default:
		return "pending"
	}
}

func NormalizeWebhookStatus(notification WebhookNotification) string {
	return normalizeProviderStatus(notification.TransactionStatus, notification.FraudStatus)
}
