package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

const mwSecret = "middleware-secret"

func protectedEngine(extra ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handlers := append([]gin.HandlerFunc{Middleware(mwSecret)}, extra...)
	handlers = append(handlers, func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID, "role": claims.Role})
	})
	engine.GET("/p", handlers...)
	return engine
}

func get(engine *gin.Engine, authz string) int {
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

func mustToken(t *testing.T, secret string, userID uint, role string, ttl time.Duration) string {
	t.Helper()
	tok, err := NewToken(secret, userID, role, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestMiddleware(t *testing.T) {
	engine := protectedEngine()
	valid := mustToken(t, mwSecret, 3, "user", time.Hour)
	cases := []struct {
		name  string
		authz string
		want  int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"bad scheme", "Basic " + valid, http.StatusUnauthorized},
		{"no token part", "Bearer", http.StatusUnauthorized},
		{"garbage token", "Bearer not.a.jwt", http.StatusUnauthorized},
		{"expired token", "Bearer " + mustToken(t, mwSecret, 3, "user", -time.Minute), http.StatusUnauthorized},
		{"wrong secret", "Bearer " + mustToken(t, "other-secret", 3, "user", time.Hour), http.StatusUnauthorized},
		{"valid", "Bearer " + valid, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := get(engine, tc.authz); got != tc.want {
				t.Fatalf("status %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMiddlewareRejectsNoneAlgorithm(t *testing.T) {
	claims := &Claims{UserID: 1, Role: "admin", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if got := get(protectedEngine(), "Bearer "+unsigned); got != http.StatusUnauthorized {
		t.Fatalf("alg=none token accepted: %d", got)
	}
}

func TestRequireRole(t *testing.T) {
	engine := protectedEngine(RequireRole("admin"))
	if got := get(engine, "Bearer "+mustToken(t, mwSecret, 1, "user", time.Hour)); got != http.StatusForbidden {
		t.Fatalf("user on admin route: %d, want 403", got)
	}
	if got := get(engine, "Bearer "+mustToken(t, mwSecret, 1, "admin", time.Hour)); got != http.StatusOK {
		t.Fatalf("admin on admin route: %d, want 200", got)
	}

	// Without the auth middleware there are no claims at all.
	gin.SetMode(gin.TestMode)
	bare := gin.New()
	bare.GET("/p", RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })
	if got := get(bare, ""); got != http.StatusForbidden {
		t.Fatalf("missing claims: %d, want 403", got)
	}
	wrongType := gin.New()
	wrongType.GET("/p", func(c *gin.Context) { c.Set(UserContextKey, "not-claims") }, RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })
	if got := get(wrongType, ""); got != http.StatusForbidden {
		t.Fatalf("invalid claims type: %d, want 403", got)
	}
}

func TestTokenRoundTrip(t *testing.T) {
	tok := mustToken(t, mwSecret, 42, "provider", time.Hour)
	claims, err := ParseToken(mwSecret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.Role != "provider" {
		t.Fatalf("claims = %+v", claims)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Fatalf("expiry not in the future: %v", claims.ExpiresAt)
	}
	if _, err := ParseToken("wrong", tok); err == nil {
		t.Fatal("token verified with the wrong secret")
	}
}

func TestTokenHelpers(t *testing.T) {
	a, err := generateSecureToken(32)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := generateSecureToken(32)
	if a == b || len(a) < 40 {
		t.Fatalf("weak tokens %q %q", a, b)
	}
	if hashToken("x") != hashToken("x") || hashToken("x") == hashToken("y") || len(hashToken("x")) != 64 {
		t.Fatal("hashToken must be a stable sha256 hex digest")
	}
}
