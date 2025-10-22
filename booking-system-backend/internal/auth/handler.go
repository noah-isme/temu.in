package auth

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "github.com/temu-in/temu.in/booking-system-backend/internal/user"
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
    "github.com/temu-in/temu.in/booking-system-backend/internal/config"
)

type Handler struct {
    repo   *user.Repository
    config *config.Config
}

func NewHandler(repo *user.Repository, cfg *config.Config) *Handler {
    return &Handler{repo: repo, config: cfg}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.POST("/register", h.Register)
    rg.POST("/login", h.Login)
}

type registerReq struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
    Name     string `json:"name"`
}

func (h *Handler) Register(c *gin.Context) {
    var req registerReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    existing, err := h.repo.FindByEmail(req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
        return
    }
    if existing != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "email exists"})
        return
    }

    pw, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    user := &models.User{Email: req.Email, Password: string(pw), Name: req.Name, Role: "user"}
    if err := h.repo.Create(user); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
        return
    }

    token, err := NewToken(h.config.JWTSecret, user.ID, user.Role, h.config.AccessTokenTTL)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": user.ID, "email": user.Email, "role": user.Role}})
}

type loginReq struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
    var req loginReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    u, err := h.repo.FindByEmail(req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
        return
    }
    if u == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    token, err := NewToken(h.config.JWTSecret, u.ID, u.Role, h.config.AccessTokenTTL)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": u.ID, "email": u.Email, "role": u.Role}})
}
