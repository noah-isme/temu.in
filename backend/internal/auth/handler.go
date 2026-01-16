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

	"temu.in/backend/internal/config"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/notification"
	"temu.in/backend/internal/token"
	"temu.in/backend/internal/user"
	"temu.in/backend/internal/validator"
)

type Handler struct {
	repo            *user.Repository
	config          *config.Config
	tokens          *token.Repository
	notificationSvc notification.Service
}

func NewHandler(repo *user.Repository, cfg *config.Config, tokens *token.Repository, notificationSvc notification.Service) *Handler {
	return &Handler{repo: repo, config: cfg, tokens: tokens, notificationSvc: notificationSvc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
	rg.POST("/logout", h.Logout)
	rg.POST("/forgot-password", h.ForgotPassword)
	rg.POST("/reset-password", h.ResetPassword)
	rg.POST("/verify-email", h.VerifyEmail)
}

type verifyEmailReq struct {
	Token string `json:"token" binding:"required"`
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailReq
	if !validator.BindAndValidate(c, &req) {
		return
	}

	u, err := h.repo.FindByVerificationToken(req.Token)
	if err != nil || u == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	if err := h.repo.MarkVerified(u.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

type forgotPasswordReq struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordReq
	if !validator.BindAndValidate(c, &req) {
		return
	}

	u, err := h.repo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if u == nil {
		c.JSON(http.StatusOK, gin.H{"message": "If email exists, reset link sent"})
		return
	}

	tokenStr, err := generateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if err := h.repo.SavePasswordReset(u.ID, tokenStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if h.notificationSvc != nil {
		go h.notificationSvc.SendPasswordReset(c.Request.Context(), u, tokenStr)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If email exists, reset link sent"})
}

type resetPasswordReq struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordReq
	if !validator.BindAndValidate(c, &req) {
		return
	}

	userID, err := h.repo.VerifyPasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	pw, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err := h.repo.UpdatePassword(userID, string(pw)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	_ = h.repo.DeletePasswordResetToken(req.Token)

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if !validator.BindAndValidate(c, &req) {
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

	// Send verification email
	verificationToken, err := generateSecureToken(32)
	if err == nil {
		if err := h.repo.SaveVerificationToken(user.ID, verificationToken); err == nil {
			if h.notificationSvc != nil {
				go h.notificationSvc.SendEmailVerification(c.Request.Context(), user, verificationToken)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": user.ID, "email": user.Email, "role": user.Role}})
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if !validator.BindAndValidate(c, &req) {
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
