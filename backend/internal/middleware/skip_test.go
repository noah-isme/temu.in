package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/testutil"
)

func TestHealthProbesBypassRateLimiter(t *testing.T) {
	client, _ := testutil.Redis(t)
	limiter := NewRateLimiter(client, 1, time.Minute)

	r := gin.New()
	r.Use(SkipPaths(HealthPathPrefixes, limiter.Middleware()))
	r.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/providers", func(c *gin.Context) { c.Status(http.StatusOK) })

	get := func(path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code
	}

	for i := 0; i < 5; i++ {
		if code := get("/health/ready"); code != http.StatusOK {
			t.Fatalf("/health/ready attempt %d: %d", i, code)
		}
		if code := get("/api/health/ready"); code != http.StatusOK {
			t.Fatalf("/api/health/ready attempt %d: %d", i, code)
		}
	}
	if code := get("/api/providers"); code != http.StatusOK {
		t.Fatalf("first API request: %d", code)
	}
	if code := get("/api/providers"); code != http.StatusTooManyRequests {
		t.Fatalf("second API request should be limited, got %d", code)
	}
}
