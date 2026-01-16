package favorite

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/auth"
)

type Handlers struct {
	service Service
}

func NewHandlers(service Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) Register(rg *gin.RouterGroup) {
	rg.GET("/", h.List)
	rg.POST("/toggle", h.Toggle)
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

	favs, err := h.service.ListFavorites(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list favorites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": favs})
}

type toggleRequest struct {
	ServiceID  *string `json:"serviceId"`
	ProviderID *string `json:"providerId"`
}

func (h *Handlers) Toggle(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	var req toggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	added, err := h.service.ToggleFavorite(claims.UserID, req.ServiceID, req.ProviderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	message := "removed from favorites"
	if added {
		message = "added to favorites"
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "added": added})
}
