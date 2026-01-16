package payment

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	repo *Repository
	// finalizeBooking is an optional callback used to update booking status when payment confirmed
	finalizeBooking func(paymentID uint, status string) error
}

func NewHandlers(r *Repository) *Handlers {
	return &Handlers{repo: r}
}

// NewHandlersWithFinalizer allows passing a callback to finalize bookings on payment confirmation
func NewHandlersWithFinalizer(r *Repository, finalizer func(uint, string) error) *Handlers {
	return &Handlers{repo: r, finalizeBooking: finalizer}
}

func (h *Handlers) Register(rg *gin.RouterGroup) {
	rg.POST("/payments/create-intent", h.CreateIntent)
	rg.POST("/payments/confirm", h.Confirm)
}

func (h *Handlers) CreateIntent(c *gin.Context) {
	var body struct {
		Amount   int64  `json:"amount" binding:"required,gte=0"`
		Currency string `json:"currency" binding:"required"`
	}
	if !validator.BindAndValidate(c, &body) {
		return
	}
	intent := &models.Payment{
		IntentID:     "pi_" + time.Now().Format("20060102150405"),
		ClientSecret: "cs_" + time.Now().Format("150405"),
		Amount:       body.Amount,
		Currency:     body.Currency,
		Status:       "created",
	}
	if err := h.repo.Create(intent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, intent)
}

func (h *Handlers) Confirm(c *gin.Context) {
	var body struct {
		ClientSecret string `json:"client_secret" binding:"required"`
	}
	if !validator.BindAndValidate(c, &body) {
		return
	}
	p, err := h.repo.FindByIntent(body.ClientSecret)
	if err != nil || p == nil {
		p, err = h.repo.FindByClientSecret(body.ClientSecret)
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// mark succeeded unless client_secret contains 'fail'
	if body.ClientSecret == "fail" {
		p.Status = "failed"
	} else {
		p.Status = "succeeded"
	}
	if err := h.repo.Update(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	// if succeeded, and booking linked, mark booking confirmed via finalizer
	if p.Status == "succeeded" && h.finalizeBooking != nil && p.ID != 0 {
		_ = h.finalizeBooking(p.ID, "confirmed")
	}
	c.JSON(http.StatusOK, gin.H{"status": p.Status, "transactionId": "tx_" + time.Now().Format("20060102150405")})
}
