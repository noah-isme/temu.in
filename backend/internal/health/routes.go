package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	h := r.Group("/health")
	h.GET("/live", liveness)
	h.GET("/ready", readiness)
}
