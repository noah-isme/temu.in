package service

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/provider"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	service         Service
	providerService provider.Service
}

func NewHandlers(service Service, providerService provider.Service) *Handlers {
	return &Handlers{service: service, providerService: providerService}
}

func (h *Handlers) Register(rg *gin.RouterGroup, jwtSecret string) {

	// Public routes
	rg.GET("/providers/:providerId/services", h.ListByProvider)
	rg.GET("/services", h.Search)
	rg.GET("/services/search", h.Search)

	// Protected routes (provider or admin)
	protected := rg.Group("/providers/:providerId/services")
	protected.Use(auth.Middleware(jwtSecret))
	protected.POST("/", h.Create)
	protected.PUT("/:id", h.Update)
	protected.DELETE("/:id", h.Delete)
}

func (h *Handlers) checkProviderOwnership(c *gin.Context, providerID string) bool {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return false
	}
	providerRecord, err := h.providerService.GetProvider(providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return false
	}
	if claims.Role != "admin" && providerRecord.UserID != claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return false
	}
	return true
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

func (h *Handlers) ListByProvider(c *gin.Context) {
	providerID := c.Param("providerId")
	activeOnly := c.Query("active_only") == "true"

	services, err := h.service.GetServicesByProvider(providerID, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"services": services})
}

func (h *Handlers) Search(c *gin.Context) {
	var filter ServiceFilter

	filter.Name = c.Query("name")
	filter.ProviderID = c.Query("provider_id")
	filter.Category = c.Query("category")

	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		if val, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			filter.MinPrice = &val
		}
	}
	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		if val, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			filter.MaxPrice = &val
		}
	}
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		filter.IsActive = &val
	} else {
		// Default to active only unless specified
		val := true
		filter.IsActive = &val
	}

	services, err := h.service.SearchServices(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"services": services})
}

func (h *Handlers) Create(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkProviderOwnership(c, providerID) {
		return
	}
	var req CreateServiceRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	service, err := h.service.CreateService(providerID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create service"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"service": service})
}

func (h *Handlers) Update(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkProviderOwnership(c, providerID) {
		return
	}
	id := c.Param("id")
	var req UpdateServiceRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	service, err := h.service.UpdateService(id, providerID, &req)
	if err != nil {
		if err.Error() == "unauthorized: service does not belong to provider" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "service not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"service": service})
}

func (h *Handlers) Delete(c *gin.Context) {
	providerID := c.Param("providerId")
	if !h.checkProviderOwnership(c, providerID) {
		return
	}
	id := c.Param("id")

	err := h.service.DeleteService(id, providerID)
	if err != nil {
		if err.Error() == "unauthorized: service does not belong to provider" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "service not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "service deleted"})
}
