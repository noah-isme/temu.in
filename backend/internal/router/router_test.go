package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/config"
)

func TestRegisterAllHasNoDuplicateRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	cfg := &config.Config{JWTSecret: "test-secret"}

	New(engine, cfg, nil, nil).RegisterAll()

	seen := make(map[string]struct{})
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate route registered: %s", key)
		}
		seen[key] = struct{}{}
	}

	for _, required := range []string{
		"GET /api/providers",
		"GET /api/services",
		"GET /api/availability",
		"POST /api/bookings",
		"POST /api/payments/create-intent",
	} {
		if _, exists := seen[required]; !exists {
			t.Errorf("required route is missing: %s", required)
		}
	}
}

func TestPaymentCreationRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	cfg := &config.Config{JWTSecret: "test-secret"}
	New(engine, cfg, nil, nil).RegisterAll()

	req := httptest.NewRequest(http.MethodPost, "/api/payments/create-intent", nil)
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
