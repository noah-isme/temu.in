package review

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/validator"
)

// reviewService is the subset of *Service the handlers use.
type reviewService interface {
	CreateReview(userID uint, bookingID uint, req CreateReviewRequest) (*models.Review, error)
	ListProviderReviews(providerID string, limit, offset int) ([]models.Review, error)
	ListUserReviews(userID uint, limit, offset int) ([]models.Review, error)
	GetProviderStats(providerID string) (*ProviderReviewStats, error)
}

type Handlers struct {
	service reviewService
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) Register(rg *gin.RouterGroup) {
	// Public routes
	rg.GET("/providers/:providerId/reviews", h.ListProviderReviews)
	rg.GET("/providers/:providerId/reviews/stats", h.GetProviderStats)
}

func (h *Handlers) RegisterProtected(rg *gin.RouterGroup) {
	rg.POST("/bookings/:bookingId/reviews", h.CreateReview)
	rg.GET("/reviews/me", h.ListUserReviews)
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

func (h *Handlers) CreateReview(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	bookingIDStr := c.Param("bookingId")
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	var req CreateReviewRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	review, err := h.service.CreateReview(claims.UserID, uint(bookingID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"review": NewPublicReview(review)})
}

func (h *Handlers) ListProviderReviews(c *gin.Context) {
	providerID := c.Param("providerId")

	limit := 10
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

	reviews, err := h.service.ListProviderReviews(providerID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reviews": newPublicReviews(reviews)})
}

func (h *Handlers) ListUserReviews(c *gin.Context) {
	claims, ok := h.getUserClaims(c)
	if !ok {
		return
	}

	limit := 10
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

	reviews, err := h.service.ListUserReviews(claims.UserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reviews": newPublicReviews(reviews)})
}

func (h *Handlers) GetProviderStats(c *gin.Context) {
	providerID := c.Param("providerId")
	stats, err := h.service.GetProviderStats(providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
