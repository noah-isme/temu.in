package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/config"
)

func TestServerMiddlewareRequestIDAndRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New(&config.Config{AllowedOrigins: []string{"http://localhost:5173"}})
	s.router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	s.router.GET("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w.Code != http.StatusNoContent || w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("ok: code=%d request id=%q", w.Code, w.Header().Get("X-Request-ID"))
	}

	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError || w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("panic: code=%d request id=%q", w.Code, w.Header().Get("X-Request-ID"))
	}
}
