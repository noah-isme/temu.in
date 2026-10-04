package booking

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/availability"
	paymentpkg "temu.in/backend/internal/payment"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	service         *Service
	providerService ProviderService
}

func NewHandlers(repo *Repository, paymentRepo *paymentpkg.Repository, userRepo UserRepo, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service:         NewService(repo, paymentRepo, userRepo, providerSvc),
		providerService: providerSvc,
	}
}

func NewHandlersWithAvailability(repo *Repository, paymentRepo *paymentpkg.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service:         NewServiceWithDeps(repo, paymentRepo, availabilitySvc, notificationSvc, userRepo, providerSvc),
		providerService: providerSvc,
	}
}

func NewHandlersWithDependencies(repo *Repository, paymentRepo *paymentpkg.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService, serviceCatalog ServiceCatalog, paymentCreator PaymentCreator) *Handlers {
	return &Handlers{
		service:         NewServiceWithFullDeps(repo, paymentRepo, availabilitySvc, notificationSvc, userRepo, providerSvc, serviceCatalog, paymentCreator),
		providerService: providerSvc,
	}
}

func NewHandlersWithService(svc *Service, providerSvc ProviderService) *Handlers {
	return &Handlers{
		service:         svc,
		providerService: providerSvc,
	}
}

// Service exposes the underlying service so the router can attach optional
// collaborators (e.g. WithSlotInvalidator) without a wider constructor.
func (h *Handlers) Service() *Service { return h.service }

func (h *Handlers) Register(rg *gin.RouterGroup) {
	rg.GET("/bookings", h.List)
	rg.POST("/bookings", h.Create)
	rg.GET("/bookings/:bookingId", h.GetByID)
	rg.POST("/bookings/:bookingId/cancel", h.Cancel)
	rg.POST("/bookings/:bookingId/confirm", h.Confirm)
	rg.POST("/bookings/:bookingId/complete", h.Complete)
	rg.PUT("/bookings/:bookingId", h.Reschedule)

	rg.GET("/providers/:providerId/stats", h.GetProviderStats)
	rg.GET("/providers/:providerId/bookings", h.ListProviderBookings)
}

// ListProviderBookings returns the provider's bookings to its owner or an
// admin. Query: status (optional), limit, offset.
func (h *Handlers) ListProviderBookings(c *gin.Context) {
	providerID := c.Param("providerId")
	if !auth.RequireProviderAccess(c, providerID, h.providerService) {
		return
	}
	limit, offset := pageParams(c)

	bookings, err := h.service.ListProviderBookings(c.Request.Context(), providerID, c.Query("status"), limit, offset)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status filter"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": newProviderBookingDTOs(bookings)})
}

// pageParams reads limit/offset query params; unparseable values fall back to
// the defaults and the service clamps the range.
func pageParams(c *gin.Context) (int, int) {
	limit, offset := 20, 0
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
	return limit, offset
}

func (h *Handlers) GetProviderStats(c *gin.Context) {
	providerID := c.Param("providerId")
	if !auth.RequireProviderAccess(c, providerID, h.providerService) {
		return
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

	limit, offset := pageParams(c)

	bookings, err := h.service.ListUserBookings(c.Request.Context(), claims.UserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": newCustomerBookingDTOs(bookings)})
}

type createBookingRequest struct {
	ProviderID   string                `json:"providerId" binding:"required"`
	ServiceID    string                `json:"serviceId" binding:"required"`
	CustomerName string                `json:"customerName" binding:"required,max=255"`
	Slot         availability.TimeSlot `json:"slot" binding:"required"`
}

func parseTimeSlot(slot availability.TimeSlot) (time.Time, time.Time, time.Time, error) {
	if slot.Date == "" || slot.StartTime == "" || slot.EndTime == "" {
		return time.Time{}, time.Time{}, time.Time{}, errors.New("slot date and time are required")
	}
	date, err := time.Parse("2006-01-02", slot.Date)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, errors.New("invalid slot date")
	}
	start, err := time.Parse("15:04", slot.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, errors.New("invalid slot start time")
	}
	end, err := time.Parse("15:04", slot.EndTime)
	if err != nil || !start.Before(end) {
		return time.Time{}, time.Time{}, time.Time{}, errors.New("invalid slot end time")
	}
	return date, start, end, nil
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

	bookingDate, startTime, endTime, err := parseTimeSlot(req.Slot)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := CreateBookingInput{
		UserID:       claims.UserID,
		ProviderID:   req.ProviderID,
		ServiceID:    &req.ServiceID,
		CustomerName: req.CustomerName,
		BookingDate:  bookingDate,
		StartTime:    startTime,
		EndTime:      endTime,
		Notes:        "", // TODO: add notes field
	}

	booking, payment, err := h.service.CreateBooking(c.Request.Context(), input)
	if err != nil {
		if err == ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrServiceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == ErrSlotUnavailable || err == ErrBookingConflict {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrPaymentUnavailable) || errors.Is(err, paymentpkg.ErrProviderNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment provider is unavailable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	dto := NewCustomerBookingDTO(booking)
	if payment != nil {
		c.JSON(http.StatusCreated, gin.H{"booking": dto, "payment": payment})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"booking": dto})
}

func (h *Handlers) GetByID(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("bookingId")
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
	c.JSON(http.StatusOK, gin.H{"booking": NewCustomerBookingDTO(booking)})
}

type cancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *Handlers) Cancel(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("bookingId")
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
	if !h.canManageProviderBooking(c) {
		return
	}
	idStr := c.Param("bookingId")
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
	Slot availability.TimeSlot `json:"slot" binding:"required"`
}

func (h *Handlers) Reschedule(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	idStr := c.Param("bookingId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req rescheduleBookingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	bookingDate, startTime, endTime, err := parseTimeSlot(req.Slot)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.RescheduleBooking(c.Request.Context(), uint(id), claims.UserID, bookingDate, startTime, endTime)
	if err != nil {
		if errors.Is(err, ErrBookingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		if errors.Is(err, ErrServiceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrSlotUnavailable) || errors.Is(err, ErrBookingConflict) || errors.Is(err, ErrNotReschedulable) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reschedule booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking rescheduled"})
}

func (h *Handlers) Complete(c *gin.Context) {
	if !h.canManageProviderBooking(c) {
		return
	}
	idStr := c.Param("bookingId")
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

func (h *Handlers) canManageProviderBooking(c *gin.Context) bool {
	id, err := strconv.ParseUint(c.Param("bookingId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return false
	}
	booking, err := h.service.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return false
	}
	return auth.RequireProviderAccess(c, booking.ProviderID, h.providerService)
}
