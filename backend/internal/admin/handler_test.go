package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/audit"
	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
	"temu.in/backend/internal/user"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

const adminSecret = "admin-secret"

func adminEngine(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h.RegisterRoutes(engine.Group("/api"), adminSecret)
	return engine
}

func call(t *testing.T, engine *gin.Engine, method, path string, actor *models.User, payload any) (int, map[string]any) {
	t.Helper()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if actor != nil {
		tok, err := auth.NewToken(adminSecret, actor.ID, actor.Role, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// Validation and role checks run before any repository call, so they need no DB.
func TestPromoteRejectsBeforeRepository(t *testing.T) {
	engine := adminEngine(NewHandler(nil, nil))
	admin := &models.User{ID: 1, Role: "admin"}
	if code, _ := call(t, engine, http.MethodPost, "/api/admin/promote", admin, map[string]string{"email": "not-an-email"}); code != http.StatusBadRequest {
		t.Fatalf("invalid email: %d, want 400", code)
	}
	if code, _ := call(t, engine, http.MethodPost, "/api/admin/promote", &models.User{ID: 2, Role: "user"}, map[string]string{"email": "a@example.test"}); code != http.StatusForbidden {
		t.Fatalf("non-admin: %d, want 403", code)
	}
	if code, _ := call(t, engine, http.MethodGet, "/api/admin/users", nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d, want 401", code)
	}
}

func TestAdminHandlerIntegration(t *testing.T) {
	db := testutil.DB(t)
	engine := adminEngine(NewHandler(user.NewRepository(db), audit.NewRepository(db)))
	actor := testutil.CreateUser(t, db, "admin")
	target := testutil.CreateUser(t, db, "user")

	code, body := call(t, engine, http.MethodPost, "/api/admin/promote", actor, map[string]string{"email": target.Email})
	if code != http.StatusOK || body["status"] != "promoted" {
		t.Fatalf("promote: %d %v", code, body)
	}
	var promoted models.User
	if err := db.First(&promoted, target.ID).Error; err != nil || promoted.Role != "admin" {
		t.Fatalf("role after promote = %q (%v)", promoted.Role, err)
	}
	var entry models.AdminAudit
	if err := db.Where("action = ?", "promote_user").First(&entry).Error; err != nil {
		t.Fatalf("audit row missing: %v", err)
	}
	if entry.ActorID != actor.ID || entry.Target != "user:"+target.Email {
		t.Fatalf("audit row = %+v", entry)
	}

	if code, _ := call(t, engine, http.MethodPost, "/api/admin/promote", actor, map[string]string{"email": "ghost@example.test"}); code != http.StatusNotFound {
		t.Fatalf("unknown email: %d, want 404", code)
	}

	code, body = call(t, engine, http.MethodGet, "/api/admin/users", actor, nil)
	if code != http.StatusOK {
		t.Fatalf("list users: %d", code)
	}
	if users, _ := body["users"].([]any); len(users) != 2 {
		t.Fatalf("users = %v", body["users"])
	}

	code, body = call(t, engine, http.MethodGet, "/api/admin/audit", actor, nil)
	if code != http.StatusOK {
		t.Fatalf("list audit: %d", code)
	}
	if entries, _ := body["audit"].([]any); len(entries) != 1 {
		t.Fatalf("audit = %v", body["audit"])
	}
}
