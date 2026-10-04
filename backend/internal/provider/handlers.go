package provider

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/validator"
)

type Handlers struct {
	service Service
}

func NewHandlers(service Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) Register(rg *gin.RouterGroup, jwtSecret string) {
	// Public routes
	rg.GET("/providers", h.List)
	rg.GET("/providers/:providerId", h.GetByID)

	// Protected routes (provider or admin)
	protected := rg.Group("/")
	protected.Use(auth.Middleware(jwtSecret))
	protected.GET("/providers/me", h.Me)
	protected.POST("/providers", h.Create)
	protected.PUT("/providers/:providerId", h.Update)
	protected.DELETE("/providers/:providerId", h.Delete)
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
	filters := ProviderFilters{
		BusinessType: c.Query("business_type"),
		Search:       c.Query("search"),
		Page:         1,
		Limit:        20,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filters.Page = p
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filters.Limit = l
		}
	}
	if isActive := c.Query("is_active"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			filters.IsActive = &active
		}
	}

	providers, total, err := h.service.ListProviders(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": newPublicProviders(providers),
		"pagination": gin.H{
			"page":  filters.Page,
			"limit": filters.Limit,
			"total": total,
		},
	})
}

func (h *Handlers) GetByID(c *gin.Context) {
	id := c.Param("providerId")
	provider, err := h.service.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": NewPublicProvider(provider)})
}

// Me returns the provider profile owned by the authenticated user.
func (h *Handlers) Me(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}
	provider, err := h.service.GetProviderByUserID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get provider"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": NewPublicProvider(provider)})
}

func (h *Handlers) Create(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	var req CreateProviderRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	provider, err := h.service.CreateProvider(claims.UserID, &req)
	if err != nil {
		if err.Error() == "user already has a provider profile" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"provider": NewPublicProvider(provider)})
}

func (h *Handlers) Update(c *gin.Context) {
	id := c.Param("providerId")
	if !auth.RequireProviderAccess(c, id, h.service) {
		return
	}
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req UpdateProviderRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	provider, err := h.service.UpdateProvider(id, claims.UserID, claims.Role == "admin", &req)
	if err != nil {
		if err.Error() == "unauthorized: provider does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "provider not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": NewPublicProvider(provider)})
}

func (h *Handlers) Delete(c *gin.Context) {
	id := c.Param("providerId")
	if !auth.RequireProviderAccess(c, id, h.service) {
		return
	}
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	err := h.service.DeleteProvider(id, claims.UserID, claims.Role == "admin")
	if err != nil {
		if err.Error() == "unauthorized: provider does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "provider not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider deleted"})
}
