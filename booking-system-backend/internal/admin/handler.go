package admin

import (
    "net/http"

    "github.com/gin-gonic/gin"
    auth "github.com/temu-in/temu.in/booking-system-backend/internal/auth"
    "github.com/temu-in/temu.in/booking-system-backend/internal/user"
    "github.com/temu-in/temu.in/booking-system-backend/internal/audit"
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
)

type Handler struct {
    repo  *user.Repository
    audit *audit.Repository
}

func NewHandler(repo *user.Repository, auditRepo *audit.Repository) *Handler {
    return &Handler{repo: repo, audit: auditRepo}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, secret string) {
    grp := rg.Group("/admin")
    grp.Use(auth.Middleware(secret), auth.RequireRole("admin"))
    grp.POST("/promote", h.Promote)
    grp.GET("/users", h.ListUsers)
    grp.GET("/audit", h.ListAudit)
}

func (h *Handler) ListUsers(c *gin.Context) {
    users, err := h.repo.ListAll()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) ListAudit(c *gin.Context) {
    audits, err := h.audit.ListAll()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"audit": audits})
}

type promoteReq struct {
    Email string `json:"email" binding:"required,email"`
}

func (h *Handler) Promote(c *gin.Context) {
    var req promoteReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    u, err := h.repo.FindByEmail(req.Email)
    if err != nil || u == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    if err := h.repo.SetRole(u.ID, "admin"); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to promote"})
        return
    }

    // record audit (actor from JWT claims if present)
    actorID := uint(0)
    if v, ok := c.Get(auth.UserContextKey); ok {
        if claims, ok := v.(*auth.Claims); ok {
            actorID = claims.UserID
        }
    }
    _ = h.audit.Create(&models.AdminAudit{ActorID: actorID, Action: "promote_user", Target: "user:" + u.Email, Details: "promoted to admin"})

    c.JSON(http.StatusOK, gin.H{"status": "promoted"})
}
