package payment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/validator"
)

type ServiceCatalog interface {
	GetService(id string) (*models.Service, error)
}

type BookingFinalizer func(paymentID uint, providerStatus string) error

type Handlers struct {
	repo            *Repository
	service         *Service
	serviceCatalog  ServiceCatalog
	finalizeBooking BookingFinalizer
}

func NewHandlers(r *Repository) *Handlers {
	return &Handlers{repo: r}
}

// NewHandlersWithFinalizer is kept for callers that only need to wire booking
// state updates. Production uses NewHandlersWithDependencies so all payment
// operations go through the configured provider.
func NewHandlersWithFinalizer(r *Repository, finalizer BookingFinalizer) *Handlers {
	return &Handlers{repo: r, finalizeBooking: finalizer}
}

func NewHandlersWithDependencies(r *Repository, paymentService *Service, serviceCatalog ServiceCatalog, finalizer BookingFinalizer) *Handlers {
	return &Handlers{
		repo:            r,
		service:         paymentService,
		serviceCatalog:  serviceCatalog,
		finalizeBooking: finalizer,
	}
}

// Register is mounted on an authenticated router group. Webhooks are
// registered separately because they authenticate with Midtrans signatures,
// not with an end-user JWT.
func (h *Handlers) Register(rg *gin.RouterGroup) {
	rg.POST("/payments/create-intent", h.CreateIntent)
	rg.POST("/payments/confirm", h.Confirm)
}

func (h *Handlers) RegisterWebhook(rg *gin.RouterGroup) {
	rg.POST("/payments/webhook", h.Webhook)
}

func (h *Handlers) getUserClaims(c *gin.Context) (*auth.Claims, bool) {
	v, ok := c.Get(auth.UserContextKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid claims"})
		return nil, false
	}
	return claims, true
}

type createIntentRequest struct {
	ProviderID string `json:"providerId" binding:"required"`
	ServiceID  string `json:"serviceId" binding:"required"`
	Currency   string `json:"currency"`
}

func (h *Handlers) CreateIntent(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}
	if h.service == nil || h.serviceCatalog == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment provider is unavailable"})
		return
	}

	var body createIntentRequest
	if !validator.BindAndValidate(c, &body) {
		return
	}

	service, err := h.serviceCatalog.GetService(body.ServiceID)
	if err != nil || service == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if !service.IsActive || service.ProviderID != body.ProviderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service is not available for this provider"})
		return
	}
	amount, err := serviceAmount(service.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service has an invalid price"})
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if currency == "" {
		currency = "IDR"
	}
	if currency != "IDR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only IDR payments are supported"})
		return
	}

	p, err := h.service.Prepare(c.Request.Context(), CreateRequest{
		UserID:      claims.UserID,
		ProviderID:  body.ProviderID,
		ServiceID:   body.ServiceID,
		Amount:      amount,
		Currency:    currency,
		Description: service.Name,
	})
	if err != nil {
		h.writeProviderError(c, err)
		return
	}
	if err := h.repo.Create(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

type confirmRequest struct {
	ClientSecret string `json:"client_secret"`
	OrderID      string `json:"order_id"`
}

func (h *Handlers) Confirm(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment provider is unavailable"})
		return
	}

	var body confirmRequest
	if !validator.BindAndValidate(c, &body) {
		return
	}
	if body.ClientSecret == "" && body.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_secret or order_id is required"})
		return
	}

	p, err := h.findPayment(body.OrderID, body.ClientSecret)
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if p.UserID != claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	transaction, err := h.service.GetTransaction(c.Request.Context(), p.OrderID)
	if err != nil {
		h.writeProviderError(c, err)
		return
	}
	status := transaction.Status
	if status == "" {
		status = models.PaymentStatusPending
	}
	updated, err := h.repo.UpdateProviderStatus(p.ID, status, transaction.TransactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if updated != nil {
		p = updated
	}
	if err := h.finalize(c, p.ID, p.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payment state update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":        p.Status,
		"transactionId": p.TransactionID,
		"order_id":      p.OrderID,
	})
}

func (h *Handlers) Webhook(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment provider is unavailable"})
		return
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook body"})
		return
	}
	var notification WebhookNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook body"})
		return
	}
	if !h.service.VerifyWebhook(notification) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	p, err := h.repo.FindByOrderID(notification.OrderID)
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if notification.GrossAmount != "" {
		grossAmount, parseErr := strconv.ParseFloat(notification.GrossAmount, 64)
		if parseErr != nil || grossAmount < 0 || int64(math.Round(grossAmount)) != p.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "webhook amount does not match payment"})
			return
		}
	}
	status := NormalizeWebhookStatus(notification)
	updated, err := h.repo.UpdateProviderStatus(p.ID, status, notification.TransactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if updated != nil {
		p = updated
	}
	if err := h.finalize(c, p.ID, p.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "booking state update failed"})
		return
	}

	digest := sha256.Sum256(raw)
	claimed, err := h.repo.RecordWebhookEvent(hex.EncodeToString(digest[:]), p.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "record webhook failed"})
		return
	}
	if !claimed {
		c.JSON(http.StatusOK, gin.H{"status": "duplicate"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (h *Handlers) findPayment(orderID, clientSecret string) (*models.Payment, error) {
	if orderID != "" {
		if p, err := h.repo.FindByOrderID(orderID); err == nil {
			return p, nil
		}
	}
	if clientSecret != "" {
		return h.repo.FindByClientSecret(clientSecret)
	}
	return nil, errors.New("payment not found")
}

func (h *Handlers) finalize(c *gin.Context, paymentID uint, status string) error {
	if h.finalizeBooking == nil {
		return nil
	}
	if status != models.PaymentStatusSucceeded && status != models.PaymentStatusFailed && status != models.PaymentStatusExpired && status != models.PaymentStatusCancelled && status != models.PaymentStatusPending {
		return nil
	}
	return h.finalizeBooking(paymentID, status)
}

func (h *Handlers) writeProviderError(c *gin.Context, err error) {
	if errors.Is(err, ErrProviderNotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment provider is unavailable"})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": "payment provider request failed"})
}

func serviceAmount(price float64) (int64, error) {
	if price < 0 || math.IsNaN(price) || math.IsInf(price, 0) || price > float64(math.MaxInt64) {
		return 0, errors.New("invalid price")
	}
	amount := int64(math.Round(price))
	if amount <= 0 {
		return 0, errors.New("price must be positive")
	}
	return amount, nil
}
