package user

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/temu-in/temu.in/booking-system-backend/internal/auth"
    "github.com/temu-in/temu.in/booking-system-backend/internal/user"
)

type Handler struct {
    repo *user.Repository
}

func NewHandler(repo *user.Repository) *Handler {
    return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, secret string) {
    rg.GET("/me", auth.Middleware(secret), h.Me)
}

func (h *Handler) Me(c *gin.Context) {
    v, ok := c.Get(auth.UserContextKey)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
        return
    }
    claims, ok := v.(*auth.Claims)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
        return
    }

    u, err := h.repo.FindByID(claims.UserID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": u.ID, "email": u.Email, "role": u.Role, "name": u.Name}})
}
