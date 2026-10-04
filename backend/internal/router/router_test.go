package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		"GET /api/providers/me",
		"GET /api/providers/:providerId/bookings",
		"GET /health/live",
		"GET /health/ready",
		"GET /api/health/live",
		"GET /api/health/ready",
		"GET /api/openapi.yaml",
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

func TestHealthReadyWithNilDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: "test-secret"}, nil, nil).RegisterAll()

	for _, path := range []string{"/health/ready", "/api/health/ready"} {
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusOK {
			t.Fatalf("%s: got status %d, want 200", path, res.Code)
		}
		if body := res.Body.String(); !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"checks":{}`) {
			t.Fatalf("%s: unexpected body %s", path, body)
		}
	}
}

func TestOpenAPISpecIsServed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: "test-secret"}, nil, nil).RegisterAll()

	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/yaml; charset=utf-8" {
		t.Fatalf("content type %q", ct)
	}
	if !strings.HasPrefix(res.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("body does not look like the spec: %.60q", res.Body.String())
	}
}
