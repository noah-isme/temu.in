package availability

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/provider"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	service         *Service
	providerService provider.Service
	serviceCatalog  ServiceCatalog
}

type ServiceCatalog interface {
	GetService(id string) (*models.Service, error)
}

func NewHandlers(service *Service, providerService provider.Service, catalogs ...ServiceCatalog) *Handlers {
	var serviceCatalog ServiceCatalog
	if len(catalogs) > 0 {
		serviceCatalog = catalogs[0]
	}
	return &Handlers{
		service:         service,
		providerService: providerService,
		serviceCatalog:  serviceCatalog,
	}
}

func (h *Handlers) Register(rg *gin.RouterGroup, jwtSecret string) {
	// Public slot lookup used by the booking flow.
	rg.GET("/providers/:providerId/availability", h.GetAvailableSlots)
	rg.GET("/availability", h.GetAvailableSlots)

	// Protected routes (provider owner or admin)
	protected := rg.Group("/providers/:providerId")
	protected.Use(auth.Middleware(jwtSecret))

	protected.GET("/schedules", h.ListSchedules)
	protected.POST("/schedules", h.CreateSchedule)
	protected.PUT("/schedules/:id", h.UpdateSchedule)
	protected.DELETE("/schedules/:id", h.DeleteSchedule)

	protected.GET("/exceptions", h.ListExceptions)
	protected.POST("/exceptions", h.CreateException)
	protected.PUT("/exceptions/:id", h.UpdateException)
	protected.DELETE("/exceptions/:id", h.DeleteException)
}

func (h *Handlers) GetAvailableSlots(c *gin.Context) {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Query("providerId")
	}
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerId is required"})
		return
	}

	dateText := c.Query("date")
	if dateText == "" {
		dateText = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	serviceID := c.Query("serviceId")
	if serviceID == "" {
		serviceID = c.Query("service_id")
	}
	if serviceID == "" || h.serviceCatalog == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceId is required"})
		return
	}
	selectedService, err := h.serviceCatalog.GetService(serviceID)
	if err != nil || selectedService == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if !selectedService.IsActive || selectedService.ProviderID != providerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service is not available for this provider"})
		return
	}

	slots, err := h.service.repo.GetAvailableSlots(providerID, date, selectedService.Duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate availability"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"date": dateText, "provider_id": providerID, "service_id": serviceID, "available_slots": slots, "slots": slots})
}

func (h *Handlers) getUserClaims(c *gin.Context) (*auth.Claims, bool) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return nil, false
	}
	return claims, true
}

func (h *Handlers) checkOwnership(c *gin.Context, providerID string) bool {
	return auth.RequireProviderAccess(c, providerID, h.providerService)
}

// Schedules

type CreateScheduleRequest struct {
	DayOfWeek   int    `json:"day_of_week" binding:"min=0,max=6"`
	StartTime   string `json:"start_time" binding:"required,len=5"` // HH:MM
	EndTime     string `json:"end_time" binding:"required,len=5"`   // HH:MM
	IsAvailable bool   `json:"is_available"`
}

func (h *Handlers) ListSchedules(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkOwnership(c, providerID) {
		return
	}

	schedules, err := h.service.GetSchedulesByProvider(providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

func (h *Handlers) CreateSchedule(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkOwnership(c, providerID) {
		return
	}

	var req CreateScheduleRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	schedule := &models.AvailabilitySchedule{
		ProviderID:  providerID,
		DayOfWeek:   req.DayOfWeek,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsAvailable: req.IsAvailable,
	}

	if err := h.service.CreateSchedule(schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

func (h *Handlers) UpdateSchedule(c *gin.Context) {
	providerID := c.Param("providerId")
	id := c.Param("id")
	if !h.checkOwnership(c, providerID) {
		return
	}

	var req CreateScheduleRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	schedule := &models.AvailabilitySchedule{
		ID:          id,
		ProviderID:  providerID,
		DayOfWeek:   req.DayOfWeek,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsAvailable: req.IsAvailable,
	}

	if err := h.service.UpdateSchedule(id, providerID, schedule); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

func (h *Handlers) DeleteSchedule(c *gin.Context) {
	providerID := c.Param("providerId")
	id := c.Param("id")
	if !h.checkOwnership(c, providerID) {
		return
	}

	if err := h.service.DeleteSchedule(id, providerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

// Exceptions

type CreateExceptionRequest struct {
	Date        string `json:"date" binding:"required"` // YYYY-MM-DD
	StartTime   string `json:"start_time"`              // Optional
	EndTime     string `json:"end_time"`                // Optional
	IsAvailable bool   `json:"is_available"`
	Reason      string `json:"reason"`
}

func (h *Handlers) ListExceptions(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkOwnership(c, providerID) {
		return
	}
	// Optional date range
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	now := time.Now()
	startDate := now
	endDate := now.AddDate(0, 3, 0) // Default 3 months

	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = t
		}
	}

	exceptions, err := h.service.GetExceptionsByProviderAndDateRange(providerID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list exceptions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"exceptions": exceptions})
}

func (h *Handlers) CreateException(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkOwnership(c, providerID) {
		return
	}

	var req CreateExceptionRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	exception := &models.AvailabilityException{
		ProviderID:  providerID,
		Date:        date,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsAvailable: req.IsAvailable,
		Reason:      req.Reason,
	}

	if err := h.service.CreateException(exception); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"exception": exception})
}

func (h *Handlers) UpdateException(c *gin.Context) {
	providerID := c.Param("providerId")
	id := c.Param("id")
	if !h.checkOwnership(c, providerID) {
		return
	}

	var req CreateExceptionRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	exception := &models.AvailabilityException{
		ID:          id,
		ProviderID:  providerID,
		Date:        date,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsAvailable: req.IsAvailable,
		Reason:      req.Reason,
	}

	if err := h.service.UpdateException(id, providerID, exception); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "exception not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exception": exception})
}

func (h *Handlers) DeleteException(c *gin.Context) {
	providerID := c.Param("providerId")
	id := c.Param("id")
	if !h.checkOwnership(c, providerID) {
		return
	}

	if err := h.service.DeleteException(id, providerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "exception not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "exception deleted"})
}
