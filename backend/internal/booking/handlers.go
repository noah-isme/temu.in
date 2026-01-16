package booking

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/payment"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	service         *Service
	providerService ProviderService
}

func NewHandlers(repo *Repository, paymentRepo *payment.Repository, userRepo UserRepo, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service:         NewService(repo, paymentRepo, userRepo, providerSvc),
		providerService: providerSvc,
	}
}

func NewHandlersWithAvailability(repo *Repository, paymentRepo *payment.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service: NewServiceWithDeps(repo, paymentRepo, availabilitySvc, notificationSvc, userRepo, providerSvc),
	}
}

func NewHandlersWithService(svc *Service, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service:         svc,
		providerService: providerSvc,
	}
}

func (h *Handlers) Register(rg *gin.RouterGroup) {
	rg.GET("/bookings", h.List)
	rg.POST("/bookings", h.Create)
	rg.GET("/bookings/:id", h.GetByID)
	rg.POST("/bookings/:id/cancel", h.Cancel)
	rg.POST("/bookings/:id/confirm", h.Confirm)
	rg.POST("/bookings/:id/complete", h.Complete)
	rg.PUT("/bookings/:id", h.Reschedule)

	rg.GET("/providers/:providerId/stats", h.GetProviderStats)
}

func (h *Handlers) GetProviderStats(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	providerID := c.Param("providerId")

	if h.providerService != nil {
		p, err := h.providerService.GetProvider(providerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		if p.UserID != strconv.Itoa(int(claims.UserID)) && claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
	}

	stats, err := h.service.GetProviderStats(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (h *Handlers) getUserClaims(c *gin.Context) (*auth.Claims, bool) {
	v, ok := c.Get("user_claims")
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

func (h *Handlers) List(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	bookings, err := h.service.ListUserBookings(c.Request.Context(), claims.UserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

type createBookingRequest struct {
	ProviderID   string `json:"providerId" binding:"required"`
	ServiceID    *uint  `json:"serviceId"`
	CustomerName string `json:"customerName"`
	Slot         string `json:"slot" binding:"required"`
	Amount       int64  `json:"amount" binding:"gte=0"`
	Currency     string `json:"currency"`
	ClientSecret string `json:"client_secret"`
	IntentID     string `json:"intent_id"`
}

func (h *Handlers) Create(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	var req createBookingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Parse slot: expected format "YYYY-MM-DD HH:MM-HH:MM"
	parts := strings.Split(req.Slot, " ")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot format"})
		return
	}
	dateStr := parts[0]
	timeRange := strings.Split(parts[1], "-")
	if len(timeRange) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot format"})
		return
	}
	startTimeStr := timeRange[0]
	endTimeStr := timeRange[1]

	bookingDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format"})
		return
	}
	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format"})
		return
	}

	input := CreateBookingInput{
		UserID:       claims.UserID,
		ProviderID:   req.ProviderID,
		ServiceID:    req.ServiceID,
		CustomerName: req.CustomerName,
		BookingDate:  bookingDate,
		StartTime:    startTime,
		EndTime:      endTime,
		Notes:        "", // TODO: add notes field
		Amount:       req.Amount,
		Currency:     req.Currency,
	}

	booking, payment, err := h.service.CreateBooking(c.Request.Context(), input)
	if err != nil {
		if err == ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err == ErrSlotUnavailable || err == ErrBookingConflict {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	if req.ClientSecret != "" || req.IntentID != "" {
		_ = h.service.LinkPaymentToBooking(c.Request.Context(), booking.ID, req.ClientSecret, req.IntentID)
	}

	if payment != nil {
		c.JSON(http.StatusCreated, gin.H{"booking": booking, "payment": payment})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"booking": booking})
}

func (h *Handlers) GetByID(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	booking, err := h.service.GetBookingByID(c.Request.Context(), uint(id), claims.UserID)
	if err != nil {
		if err == ErrBookingNotFound || err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get booking"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"booking": booking})
}

type cancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *Handlers) Cancel(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req cancelRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	err = h.service.CancelBooking(c.Request.Context(), uint(id), claims.UserID, req.Reason)
	if err != nil {
		if err == ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "booking cancelled"})
}

func (h *Handlers) Confirm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.ConfirmBooking(c.Request.Context(), uint(id))
	if err != nil {
		if err == ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "booking confirmed"})
}

type rescheduleBookingRequest struct {
	Slot string `json:"slot" binding:"required"`
}

func (h *Handlers) Reschedule(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req rescheduleBookingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	parts := strings.Split(req.Slot, " ")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot format"})
		return
	}
	dateStr := parts[0]
	timeRange := strings.Split(parts[1], "-")
	if len(timeRange) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot format"})
		return
	}
	startTimeStr := timeRange[0]
	endTimeStr := timeRange[1]

	bookingDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format"})
		return
	}
	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format"})
		return
	}

	err = h.service.RescheduleBooking(c.Request.Context(), uint(id), claims.UserID, bookingDate, startTime, endTime)
	if err != nil {
		if err == ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		if err == ErrSlotUnavailable || err == ErrBookingConflict {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reschedule booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking rescheduled"})
}

func (h *Handlers) Complete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.CompleteBooking(c.Request.Context(), uint(id))
	if err != nil {
		if err == ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "booking completed"})
}
