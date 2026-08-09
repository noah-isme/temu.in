package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"temu.in/backend/internal/models"
)

// ProviderLookup is the smallest dependency needed to authorize a provider
// scoped action. Keeping the policy here prevents each feature package from
// implementing subtly different ownership checks.
type ProviderLookup interface {
	GetProvider(id string) (*models.ServiceProvider, error)
}

// ClaimsFromContext returns the authenticated claims placed on the request by
// Middleware. It intentionally does not write a response so callers can use
// it in non-HTTP policy code as well.
func ClaimsFromContext(c *gin.Context) (*Claims, bool) {
	v, ok := c.Get(UserContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok && claims != nil
}

// RequireProviderAccess authorizes a provider owner or an administrator for a
// provider-scoped resource. The provider is loaded before the role override so
// an administrator cannot mutate a resource under a nonexistent provider.
func RequireProviderAccess(c *gin.Context, providerID string, lookup ProviderLookup) bool {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return false
	}
	if lookup == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authorization unavailable"})
		return false
	}

	provider, err := lookup.GetProvider(providerID)
	if err != nil || provider == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return false
	}
	if claims.Role == "admin" || provider.UserID == claims.UserID {
		return true
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	return false
}
