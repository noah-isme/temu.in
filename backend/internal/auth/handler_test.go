package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/config"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/notification"
	"temu.in/backend/internal/testutil"
	"temu.in/backend/internal/token"
	"temu.in/backend/internal/user"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

const handlerSecret = "handler-test-secret"

type authEnv struct {
	t      *testing.T
	db     *gorm.DB
	redis  *miniredis.Miniredis
	engine *gin.Engine
}

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.DB(t)
	client, mr := testutil.Redis(t)
	cfg := &config.Config{JWTSecret: handlerSecret, AccessTokenTTL: time.Hour, AppEnv: "test"}
	h := NewHandler(user.NewRepository(db), cfg, token.NewRepositoryWithCache(db, client), notification.NewService(notification.NewRepository(db)))
	engine := gin.New()
	h.RegisterRoutes(engine.Group("/api/auth"))
	return &authEnv{t: t, db: db, redis: mr, engine: engine}
}

type authResponse struct {
	code    int
	body    map[string]any
	cookies []*http.Cookie
}

func (r authResponse) refreshCookie() *http.Cookie {
	for _, c := range r.cookies {
		if c.Name == "refresh_token" {
			return c
		}
	}
	return nil
}

func (e *authEnv) post(path string, payload any, cookie *http.Cookie) authResponse {
	e.t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		e.t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth"+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return authResponse{code: w.Code, body: out, cookies: w.Result().Cookies()}
}

func TestRegisterLoginRefreshLogoutFlow(t *testing.T) {
	e := newAuthEnv(t)
	creds := map[string]string{"email": "flow@example.test", "password": "secret123", "name": "Flow"}

	reg := e.post("/register", creds, nil)
	if reg.code != http.StatusOK {
		t.Fatalf("register: %d %v", reg.code, reg.body)
	}
	if tok, _ := reg.body["token"].(string); tok == "" {
		t.Fatalf("register: no token in %v", reg.body)
	}
	if u, _ := reg.body["user"].(map[string]any); u["email"] != creds["email"] || u["role"] != "user" {
		t.Fatalf("register user = %v", reg.body["user"])
	}
	if reg.refreshCookie() == nil || !reg.refreshCookie().HttpOnly {
		t.Fatalf("register: missing HttpOnly refresh_token cookie")
	}
	if dup := e.post("/register", creds, nil); dup.code != http.StatusBadRequest {
		t.Fatalf("duplicate register: %d, want 400", dup.code)
	}

	if bad := e.post("/login", map[string]string{"email": creds["email"], "password": "wrong-password"}, nil); bad.code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d, want 401", bad.code)
	}
	if unknown := e.post("/login", map[string]string{"email": "nobody@example.test", "password": "x"}, nil); unknown.code != http.StatusUnauthorized {
		t.Fatalf("unknown user: %d, want 401", unknown.code)
	}
	login := e.post("/login", map[string]string{"email": creds["email"], "password": creds["password"]}, nil)
	if login.code != http.StatusOK || login.refreshCookie() == nil {
		t.Fatalf("login: %d %v", login.code, login.body)
	}
	claims, err := ParseToken(handlerSecret, login.body["token"].(string))
	if err != nil || claims.Role != "user" {
		t.Fatalf("login token: %+v %v", claims, err)
	}

	oldCookie := login.refreshCookie()
	refreshed := e.post("/refresh", nil, oldCookie)
	if refreshed.code != http.StatusOK || refreshed.body["token"] == "" {
		t.Fatalf("refresh: %d %v", refreshed.code, refreshed.body)
	}
	newCookie := refreshed.refreshCookie()
	if newCookie == nil || newCookie.Value == oldCookie.Value {
		t.Fatalf("refresh did not rotate the cookie")
	}
	if !e.redis.Exists("revoked_rt:" + hashToken(oldCookie.Value)) {
		t.Fatalf("old refresh token not blacklisted in Redis")
	}
	if replay := e.post("/refresh", nil, oldCookie); replay.code != http.StatusUnauthorized {
		t.Fatalf("replayed old refresh cookie: %d, want 401", replay.code)
	}
	if missing := e.post("/refresh", nil, nil); missing.code != http.StatusUnauthorized {
		t.Fatalf("refresh without cookie: %d, want 401", missing.code)
	}

	logout := e.post("/logout", nil, newCookie)
	if logout.code != http.StatusOK || logout.body["status"] != "logged_out" {
		t.Fatalf("logout: %d %v", logout.code, logout.body)
	}
	if c := logout.refreshCookie(); c == nil || c.Value != "" || c.Expires.After(time.Unix(1, 0)) {
		t.Fatalf("logout did not clear the cookie: %+v", c)
	}
	if after := e.post("/refresh", nil, newCookie); after.code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: %d, want 401", after.code)
	}
	// Revocation is durable in Postgres too, not only in the Redis blacklist.
	e.redis.FlushAll()
	if after := e.post("/refresh", nil, newCookie); after.code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout (DB path): %d, want 401", after.code)
	}
}

func TestRegisterValidation(t *testing.T) {
	e := newAuthEnv(t)
	if r := e.post("/register", map[string]string{"email": "not-an-email", "password": "secret123"}, nil); r.code != http.StatusBadRequest {
		t.Fatalf("invalid email: %d, want 400", r.code)
	}
	if r := e.post("/register", map[string]string{"email": "a@example.test", "password": "123"}, nil); r.code != http.StatusBadRequest {
		t.Fatalf("short password: %d, want 400", r.code)
	}
}

func TestForgotAndResetPassword(t *testing.T) {
	e := newAuthEnv(t)
	creds := map[string]string{"email": "reset@example.test", "password": "secret123"}
	if r := e.post("/register", creds, nil); r.code != http.StatusOK {
		t.Fatalf("register: %d", r.code)
	}

	unknown := e.post("/forgot-password", map[string]string{"email": "ghost@example.test"}, nil)
	known := e.post("/forgot-password", map[string]string{"email": creds["email"]}, nil)
	if unknown.code != http.StatusOK || known.code != http.StatusOK || unknown.body["message"] != known.body["message"] {
		t.Fatalf("forgot-password must not reveal account existence: %d %v / %d %v", unknown.code, unknown.body, known.code, known.body)
	}

	var resetToken string
	if err := e.db.Raw(`SELECT token_hash FROM password_resets ORDER BY created_at DESC LIMIT 1`).Scan(&resetToken).Error; err != nil || resetToken == "" {
		t.Fatalf("password_resets row missing: %v", err)
	}

	if r := e.post("/reset-password", map[string]string{"token": resetToken, "password": "newsecret456"}, nil); r.code != http.StatusOK {
		t.Fatalf("reset: %d %v", r.code, r.body)
	}
	if r := e.post("/login", map[string]string{"email": creds["email"], "password": "newsecret456"}, nil); r.code != http.StatusOK {
		t.Fatalf("login with new password: %d", r.code)
	}
	if r := e.post("/login", creds, nil); r.code != http.StatusUnauthorized {
		t.Fatalf("login with old password: %d, want 401", r.code)
	}
	if r := e.post("/reset-password", map[string]string{"token": resetToken, "password": "another789"}, nil); r.code != http.StatusBadRequest {
		t.Fatalf("reused reset token: %d, want 400", r.code)
	}
}

func TestVerifyEmail(t *testing.T) {
	e := newAuthEnv(t)
	if r := e.post("/register", map[string]string{"email": "verify@example.test", "password": "secret123"}, nil); r.code != http.StatusOK {
		t.Fatalf("register: %d", r.code)
	}
	var u models.User
	if err := e.db.Where("email = ?", "verify@example.test").First(&u).Error; err != nil {
		t.Fatal(err)
	}
	if u.IsVerified || u.VerificationToken == nil || *u.VerificationToken == "" {
		t.Fatalf("new user should be unverified with a token: %+v", u)
	}

	if r := e.post("/verify-email", map[string]string{"token": "bogus"}, nil); r.code != http.StatusBadRequest {
		t.Fatalf("bad token: %d, want 400", r.code)
	}
	if r := e.post("/verify-email", map[string]string{"token": *u.VerificationToken}, nil); r.code != http.StatusOK {
		t.Fatalf("verify: %d %v", r.code, r.body)
	}
	var after models.User
	if err := e.db.First(&after, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !after.IsVerified {
		t.Fatal("is_verified not set")
	}
	if r := e.post("/verify-email", map[string]string{"token": *u.VerificationToken}, nil); r.code != http.StatusBadRequest {
		t.Fatalf("reused verification token: %d, want 400", r.code)
	}
}
