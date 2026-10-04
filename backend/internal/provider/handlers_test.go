package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
)

// fakeService is an in-memory provider.Service for handler tests.
type fakeService struct {
	providers []models.ServiceProvider
	err       error
}

func (f *fakeService) CreateProvider(userID uint, req *CreateProviderRequest) (*models.ServiceProvider, error) {
	if f.err != nil {
		return nil, f.err
	}
	p := models.ServiceProvider{ID: fmt.Sprintf("p-%d", len(f.providers)+1), UserID: userID, BusinessName: req.BusinessName,
		BusinessType: req.BusinessType, Address: req.Address, IsActive: true,
		User: models.User{ID: userID, Email: "owner@example.test", Name: "Owner", Role: "provider", IsVerified: true}}
	f.providers = append(f.providers, p)
	return &p, nil
}

func (f *fakeService) GetProvider(id string) (*models.ServiceProvider, error) {
	if f.err != nil {
		return nil, f.err
	}
	for i := range f.providers {
		if f.providers[i].ID == id {
			p := f.providers[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("failed to get provider: %w", gorm.ErrRecordNotFound)
}

func (f *fakeService) GetProviderByUserID(userID uint) (*models.ServiceProvider, error) {
	if f.err != nil {
		return nil, f.err
	}
	for i := range f.providers {
		if f.providers[i].UserID == userID {
			p := f.providers[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("failed to get provider: %w", gorm.ErrRecordNotFound)
}

func (f *fakeService) UpdateProvider(id string, userID uint, isAdmin bool, req *UpdateProviderRequest) (*models.ServiceProvider, error) {
	p, err := f.GetProvider(id)
	if err != nil {
		return nil, errors.New("provider not found")
	}
	if req.BusinessName != nil {
		p.BusinessName = *req.BusinessName
	}
	return p, nil
}

func (f *fakeService) DeleteProvider(id string, userID uint, isAdmin bool) error { return f.err }

func (f *fakeService) ListProviders(filters ProviderFilters) ([]models.ServiceProvider, int64, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.providers, int64(len(f.providers)), nil
}

const testSecret = "test-secret"

func newTestEngine(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandlers(svc).Register(engine.Group("/api"), testSecret)
	return engine
}

func request(t *testing.T, engine *gin.Engine, method, path string, userID uint, role string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if userID != 0 {
		token, err := auth.NewToken(testSecret, userID, role, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	var body map[string]interface{}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return res, body
}

func ownerFixture() *fakeService {
	return &fakeService{providers: []models.ServiceProvider{{
		ID: "prov-1", UserID: 7, BusinessName: "Salon Satu", BusinessType: "salon", IsActive: true,
		User: models.User{ID: 7, Email: "owner@example.test", Name: "Siti", Role: "provider", IsVerified: true},
	}}}
}

func TestMe(t *testing.T) {
	engine := newTestEngine(ownerFixture())

	if res, _ := request(t, engine, http.MethodGet, "/api/providers/me", 0, ""); res.Code != http.StatusUnauthorized {
		t.Errorf("anonymous status %d, want 401", res.Code)
	}
	res, body := request(t, engine, http.MethodGet, "/api/providers/me", 8, "user")
	if res.Code != http.StatusNotFound || body["error"] != "provider not found" {
		t.Errorf("no provider: status %d body %v", res.Code, body)
	}
	res, body = request(t, engine, http.MethodGet, "/api/providers/me", 7, "provider")
	if res.Code != http.StatusOK {
		t.Fatalf("owner status %d: %s", res.Code, res.Body.String())
	}
	p := body["provider"].(map[string]interface{})
	if p["id"] != "prov-1" || p["business_name"] != "Salon Satu" {
		t.Errorf("provider = %v", p)
	}

	broken := newTestEngine(&fakeService{err: errors.New("db down")})
	if res, _ := request(t, broken, http.MethodGet, "/api/providers/me", 7, "provider"); res.Code != http.StatusInternalServerError {
		t.Errorf("service error status %d, want 500", res.Code)
	}
}

func TestMeDoesNotShadowProviderByID(t *testing.T) {
	engine := newTestEngine(ownerFixture())
	res, body := request(t, engine, http.MethodGet, "/api/providers/prov-1", 0, "")
	if res.Code != http.StatusOK || body["provider"].(map[string]interface{})["id"] != "prov-1" {
		t.Fatalf("GET /providers/prov-1 status %d body %v", res.Code, body)
	}
}

var piiKeys = []string{"email", "role", "isVerified", "password", "verification_token", "user", "user_id"}

func findKeys(v interface{}, keys []string) []string {
	var found []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			for _, bad := range keys {
				if k == bad {
					found = append(found, k)
				}
			}
			found = append(found, findKeys(child, keys)...)
		}
	case []interface{}:
		for _, child := range val {
			found = append(found, findKeys(child, keys)...)
		}
	}
	return found
}

func TestProviderResponsesExposeNoPII(t *testing.T) {
	svc := ownerFixture()
	engine := newTestEngine(svc)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		user   uint
		role   string
		status int
	}{
		{"list", http.MethodGet, "/api/providers", "", 0, "", http.StatusOK},
		{"get", http.MethodGet, "/api/providers/prov-1", "", 0, "", http.StatusOK},
		{"me", http.MethodGet, "/api/providers/me", "", 7, "provider", http.StatusOK},
		{"update", http.MethodPut, "/api/providers/prov-1", `{"business_name":"Salon Dua"}`, 7, "provider", http.StatusOK},
		{"create", http.MethodPost, "/api/providers", `{"business_name":"New","business_type":"salon","address":"Jl. 2"}`, 9, "user", http.StatusCreated},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		if tc.user != 0 {
			token, _ := auth.NewToken(testSecret, tc.user, tc.role, time.Hour)
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, res.Code, tc.status, res.Body.String())
			continue
		}
		var body interface{}
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if bad := findKeys(body, piiKeys); len(bad) > 0 {
			t.Errorf("%s: response contains %v: %s", tc.name, bad, res.Body.String())
		}
	}
}

func TestProviderListEnvelopeAndOwnerName(t *testing.T) {
	engine := newTestEngine(ownerFixture())
	res, body := request(t, engine, http.MethodGet, "/api/providers?page=2&limit=5", 0, "")
	if res.Code != http.StatusOK {
		t.Fatalf("status %d", res.Code)
	}
	pagination := body["pagination"].(map[string]interface{})
	if pagination["page"].(float64) != 2 || pagination["limit"].(float64) != 5 || pagination["total"].(float64) != 1 {
		t.Errorf("pagination = %v", pagination)
	}
	providers := body["providers"].([]interface{})
	p := providers[0].(map[string]interface{})
	for _, key := range []string{"id", "business_name", "business_type", "description", "address", "latitude", "longitude", "pricing_info", "is_active", "created_at", "updated_at"} {
		if _, ok := p[key]; !ok {
			t.Errorf("provider missing %q", key)
		}
	}
	if p["owner_name"] != "Siti" {
		t.Errorf("owner_name = %v", p["owner_name"])
	}

	empty := newTestEngine(&fakeService{})
	res, _ = request(t, empty, http.MethodGet, "/api/providers", 0, "")
	if !strings.Contains(res.Body.String(), `"providers":[]`) {
		t.Errorf("empty list body = %s", res.Body.String())
	}
}
