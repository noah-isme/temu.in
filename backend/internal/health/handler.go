package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func readiness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"checked_at": time.Now().UTC(),
	})
}

func liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}
