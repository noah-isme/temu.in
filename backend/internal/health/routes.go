package health

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts /live and /ready on rg.
func RegisterRoutes(rg gin.IRoutes, h *Handler) {
	rg.GET("/live", h.Liveness)
	rg.GET("/ready", h.Readiness)
}
