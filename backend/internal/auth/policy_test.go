package auth

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/models"
)

type policyProviderLookup struct {
	provider *models.ServiceProvider
	err      error
}

func (f policyProviderLookup) GetProvider(string) (*models.ServiceProvider, error) {
	return f.provider, f.err
}

func policyContext(claims *Claims) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	if claims != nil {
		c.Set(UserContextKey, claims)
	}
	return c, recorder
}

func TestRequireProviderAccess(t *testing.T) {
	provider := &models.ServiceProvider{ID: "provider-1", UserID: 7}
	lookup := policyProviderLookup{provider: provider}

	tests := []struct {
		name        string
		claims      *Claims
		lookup      ProviderLookup
		wantAllowed bool
		wantStatus  int
	}{
		{name: "provider owner", claims: &Claims{UserID: 7, Role: "user"}, lookup: lookup, wantAllowed: true},
		{name: "admin", claims: &Claims{UserID: 99, Role: "admin"}, lookup: lookup, wantAllowed: true},
		{name: "cross tenant user", claims: &Claims{UserID: 8, Role: "user"}, lookup: lookup, wantStatus: 403},
		{name: "customer cannot manage provider", claims: &Claims{UserID: 8, Role: "customer"}, lookup: lookup, wantStatus: 403},
		{name: "missing authentication", lookup: lookup, wantStatus: 401},
		{name: "unknown provider", claims: &Claims{UserID: 7, Role: "user"}, lookup: policyProviderLookup{err: errors.New("not found")}, wantStatus: 404},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := policyContext(tt.claims)
			allowed := RequireProviderAccess(c, "provider-1", tt.lookup)
			if allowed != tt.wantAllowed {
				t.Fatalf("allowed=%v, want %v", allowed, tt.wantAllowed)
			}
			if tt.wantStatus != 0 && recorder.Code != tt.wantStatus {
				t.Fatalf("status=%d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}
