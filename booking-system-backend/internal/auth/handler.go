package auth

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "github.com/temu-in/temu.in/booking-system-backend/internal/user"
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
    "github.com/temu-in/temu.in/booking-system-backend/internal/config"
    "github.com/temu-in/temu.in/booking-system-backend/internal/token"
)

type Handler struct {
    repo    *user.Repository
    config  *config.Config
    tokens  *token.Repository
}

func NewHandler(repo *user.Repository, cfg *config.Config, tokens *token.Repository) *Handler {
    return &Handler{repo: repo, config: cfg, tokens: tokens}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.POST("/register", h.Register)
    rg.POST("/login", h.Login)
    rg.POST("/refresh", h.Refresh)
    rg.POST("/logout", h.Logout)
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

        // create refresh token (secure random)
        rt, err := generateSecureToken(32)
        if err == nil {
            hash := hashToken(rt)
            r := &models.RefreshToken{TokenHash: hash, UserID: user.ID, ExpiresAt: time.Now().Add(24 * time.Hour)}
            _ = h.tokens.Create(r)
            cookie := &http.Cookie{Name: "refresh_token", Value: rt, HttpOnly: true, Path: "/", Expires: r.ExpiresAt}
            // security: only secure in production
            if h.config.AppEnv == "production" {
                cookie.Secure = true
                cookie.SameSite = http.SameSiteStrictMode
            } else {
                cookie.SameSite = http.SameSiteLaxMode
            }
            http.SetCookie(c.Writer, cookie)
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

        // create refresh token (secure random)
        rt, err := generateSecureToken(32)
        if err == nil {
            hash := hashToken(rt)
            r := &models.RefreshToken{TokenHash: hash, UserID: u.ID, ExpiresAt: time.Now().Add(24 * time.Hour)}
            _ = h.tokens.Create(r)
            cookie := &http.Cookie{Name: "refresh_token", Value: rt, HttpOnly: true, Path: "/", Expires: r.ExpiresAt}
            if h.config.AppEnv == "production" {
                cookie.Secure = true
                cookie.SameSite = http.SameSiteStrictMode
            } else {
                cookie.SameSite = http.SameSiteLaxMode
            }
            http.SetCookie(c.Writer, cookie)
        }

    c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": u.ID, "email": u.Email, "role": u.Role}})
}

func (h *Handler) Refresh(c *gin.Context) {
    cookie, err := c.Request.Cookie("refresh_token")
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh"})
        return
    }
    oldHash := hashToken(cookie.Value)
    rtRec, err := h.tokens.FindByHash(oldHash)
    if err != nil || rtRec == nil || rtRec.Revoked || rtRec.ExpiresAt.Before(time.Now()) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh"})
        return
    }

    u, err := h.repo.FindByID(rtRec.UserID)
    if err != nil || u == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh"})
        return
    }

    // rotate: create new refresh token, store it, revoke old
    newRT, genErr := generateSecureToken(32)
    if genErr != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh"})
        return
    }
    newHash := hashToken(newRT)
    newRec := &models.RefreshToken{TokenHash: newHash, UserID: u.ID, ExpiresAt: time.Now().Add(24 * time.Hour)}
    if err := h.tokens.Create(newRec); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh"})
        return
    }
    _ = h.tokens.RevokeByHash(oldHash)

    // set cookie for new refresh token
    cookieOut := &http.Cookie{Name: "refresh_token", Value: newRT, HttpOnly: true, Path: "/", Expires: newRec.ExpiresAt}
    if h.config.AppEnv == "production" {
        cookieOut.Secure = true
        cookieOut.SameSite = http.SameSiteStrictMode
    } else {
        cookieOut.SameSite = http.SameSiteLaxMode
    }
    http.SetCookie(c.Writer, cookieOut)

    token, err := NewToken(h.config.JWTSecret, u.ID, u.Role, h.config.AccessTokenTTL)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) Logout(c *gin.Context) {
    cookie, err := c.Request.Cookie("refresh_token")
    if err == nil {
        hash := hashToken(cookie.Value)
        _ = h.tokens.RevokeByHash(hash)
        // clear cookie
        http.SetCookie(c.Writer, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", Expires: time.Unix(0, 0)})
    }
    c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func generateSecureToken(n int) (string, error) {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(t string) string {
    h := sha256.Sum256([]byte(t))
    return hex.EncodeToString(h[:])
}
