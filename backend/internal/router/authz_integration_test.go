package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/config"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// Integration tests run the fully wired router against a per-package Postgres
// database (testutil, TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

const authzSecret = "authz-test-secret"

type authzEnv struct {
	t      *testing.T
	db     *gorm.DB
	engine *gin.Engine
}

func newAuthzEnv(t *testing.T) *authzEnv {
	t.Helper()
	db := testutil.DB(t)
	cache, _ := testutil.Redis(t)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: authzSecret}, db, cache).RegisterAll()
	return &authzEnv{t: t, db: db, engine: engine}
}

func (e *authzEnv) token(u *models.User) string {
	e.t.Helper()
	tok, err := auth.NewToken(authzSecret, u.ID, u.Role, time.Hour)
	if err != nil {
		e.t.Fatal(err)
	}
	return tok
}

// do sends a request as user (nil = anonymous) and decodes a JSON body.
func (e *authzEnv) do(method, path string, user *models.User) (int, interface{}) {
	e.t.Helper()
	return e.doJSON(method, path, user, nil)
}

// doJSON is do with an optional JSON request body.
func (e *authzEnv) doJSON(method, path string, user *models.User, payload interface{}) (int, interface{}) {
	e.t.Helper()
	var reqBody *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatal(err)
		}
		reqBody = bytes.NewReader(raw)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("Authorization", "Bearer "+e.token(user))
	}
	res := httptest.NewRecorder()
	e.engine.ServeHTTP(res, req)
	var body interface{}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return res.Code, body
}

// forbiddenKeys walks decoded JSON and returns every object key that is in
// the forbidden set.
func forbiddenKeys(v interface{}, forbidden map[string]bool) []string {
	var found []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if forbidden[k] {
				found = append(found, k)
			}
			found = append(found, forbiddenKeys(child, forbidden)...)
		}
	case []interface{}:
		for _, child := range val {
			found = append(found, forbiddenKeys(child, forbidden)...)
		}
	}
	return found
}

func TestAuthzProvidersMe(t *testing.T) {
	e := newAuthzEnv(t)
	owner := testutil.CreateUser(t, e.db, "provider")
	customer := testutil.CreateUser(t, e.db, "user")
	p := testutil.CreateProvider(t, e.db, owner.ID)

	if code, _ := e.do(http.MethodGet, "/api/providers/me", nil); code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d, want 401", code)
	}
	if code, _ := e.do(http.MethodGet, "/api/providers/me", customer); code != http.StatusNotFound {
		t.Errorf("user without provider: %d, want 404", code)
	}
	code, body := e.do(http.MethodGet, "/api/providers/me", owner)
	if code != http.StatusOK {
		t.Fatalf("owner: %d, want 200", code)
	}
	got := body.(map[string]interface{})["provider"].(map[string]interface{})
	if got["id"] != p.ID {
		t.Fatalf("owner provider id = %v, want %s", got["id"], p.ID)
	}
}

func TestAuthzProviderBookings(t *testing.T) {
	e := newAuthzEnv(t)
	owner := testutil.CreateUser(t, e.db, "provider")
	customer := testutil.CreateUser(t, e.db, "user")
	admin := testutil.CreateUser(t, e.db, "admin")
	p := testutil.CreateProvider(t, e.db, owner.ID)
	svc := testutil.CreateService(t, e.db, p.ID, 60, 0)
	testutil.CreateBooking(t, e.db, customer.ID, p.ID, &svc.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusConfirmed)

	path := "/api/providers/" + p.ID + "/bookings"
	cases := []struct {
		name string
		user *models.User
		path string
		want int
	}{
		{"anonymous", nil, path, http.StatusUnauthorized},
		{"other user", customer, path, http.StatusForbidden},
		{"unknown provider", owner, "/api/providers/00000000-0000-0000-0000-000000000000/bookings", http.StatusNotFound},
		{"owner", owner, path, http.StatusOK},
		{"admin", admin, path, http.StatusOK},
	}
	for _, tc := range cases {
		code, body := e.do(http.MethodGet, tc.path, tc.user)
		if code != tc.want {
			t.Errorf("%s: %d, want %d (%v)", tc.name, code, tc.want, body)
			continue
		}
		if code == http.StatusOK {
			list := body.(map[string]interface{})["bookings"].([]interface{})
			if len(list) != 1 {
				t.Errorf("%s: %d bookings, want 1", tc.name, len(list))
			}
			if bad := forbiddenKeys(body, map[string]bool{"email": true, "admin_notes": true}); len(bad) > 0 {
				t.Errorf("%s: forbidden keys %v", tc.name, bad)
			}
		}
	}
}

// emailValues returns every non-empty string value stored under an "email" key.
func emailValues(v interface{}) []string {
	var found []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if s, ok := child.(string); ok && k == "email" && s != "" {
				found = append(found, s)
			}
			found = append(found, emailValues(child)...)
		}
	case []interface{}:
		for _, child := range val {
			found = append(found, emailValues(child)...)
		}
	}
	return found
}

func TestPublicProviderAndReviewResponsesExposeNoPII(t *testing.T) {
	e := newAuthzEnv(t)
	owner := testutil.CreateUser(t, e.db, "provider")
	customer := testutil.CreateUser(t, e.db, "user")
	p := testutil.CreateProvider(t, e.db, owner.ID)
	svc := testutil.CreateService(t, e.db, p.ID, 60, 0)
	b := testutil.CreateBooking(t, e.db, customer.ID, p.ID, &svc.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusCompleted)
	review := &models.Review{BookingID: b.ID, ProviderID: p.ID, UserID: customer.ID, Rating: 5, Comment: "great"}
	if err := e.db.Omit("Booking", "Provider", "User").Create(review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}

	pii := map[string]bool{"email": true, "role": true, "isVerified": true, "password": true, "verification_token": true}
	for _, path := range []string{
		"/api/providers",
		"/api/providers/" + p.ID,
		"/api/providers/" + p.ID + "/reviews",
	} {
		code, body := e.do(http.MethodGet, path, nil)
		if code != http.StatusOK {
			t.Errorf("GET %s: %d", path, code)
			continue
		}
		if bad := forbiddenKeys(body, pii); len(bad) > 0 {
			t.Errorf("GET %s exposes %v", path, bad)
		}
	}

	code, body := e.do(http.MethodGet, "/api/providers/"+p.ID+"/reviews", nil)
	reviews := body.(map[string]interface{})["reviews"].([]interface{})
	if code != http.StatusOK || len(reviews) != 1 || reviews[0].(map[string]interface{})["reviewer_name"] != customer.Name {
		t.Errorf("reviews = %v", body)
	}

	// Service and favourite responses still embed a zero-valued provider.user
	// (models lack omitempty), but it must never be populated.
	for _, path := range []string{"/api/providers/" + p.ID + "/services", "/api/services"} {
		code, body := e.do(http.MethodGet, path, nil)
		if code != http.StatusOK {
			t.Errorf("GET %s: %d", path, code)
			continue
		}
		if emails := emailValues(body); len(emails) > 0 {
			t.Errorf("GET %s exposes emails %v", path, emails)
		}
	}
}

func TestAuthzScheduleCreateByRole(t *testing.T) {
	e := newAuthzEnv(t)
	owner := testutil.CreateUser(t, e.db, "provider")
	customer := testutil.CreateUser(t, e.db, "user")
	admin := testutil.CreateUser(t, e.db, "admin")
	p := testutil.CreateProvider(t, e.db, owner.ID)
	path := "/api/providers/" + p.ID + "/schedules"

	cases := []struct {
		name string
		user *models.User
		day  int
		want int
	}{
		{"anonymous", nil, 1, http.StatusUnauthorized},
		{"customer", customer, 1, http.StatusForbidden},
		{"owner", owner, 2, http.StatusCreated},
		{"admin", admin, 3, http.StatusCreated},
	}
	for _, tc := range cases {
		body := map[string]interface{}{"day_of_week": tc.day, "start_time": "09:00", "end_time": "17:00", "is_available": true}
		if code, resp := e.doJSON(http.MethodPost, path, tc.user, body); code != tc.want {
			t.Errorf("%s: %d, want %d (%v)", tc.name, code, tc.want, resp)
		}
	}
	var n int64
	e.db.Model(&models.AvailabilitySchedule{}).Where("provider_id = ?", p.ID).Count(&n)
	if n != 2 {
		t.Errorf("schedules created = %d, want 2 (owner + admin)", n)
	}
}

func TestAuthzBookingConfirmByRole(t *testing.T) {
	e := newAuthzEnv(t)
	owner := testutil.CreateUser(t, e.db, "provider")
	customer := testutil.CreateUser(t, e.db, "user")
	p := testutil.CreateProvider(t, e.db, owner.ID)
	svc := testutil.CreateService(t, e.db, p.ID, 60, 0)
	b := testutil.CreateBooking(t, e.db, customer.ID, p.ID, &svc.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusPending)
	path := "/api/bookings/" + strconv.FormatUint(uint64(b.ID), 10) + "/confirm"

	if code, _ := e.do(http.MethodPost, path, nil); code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d, want 401", code)
	}
	if code, _ := e.do(http.MethodPost, path, customer); code != http.StatusForbidden {
		t.Errorf("customer: %d, want 403", code)
	}
	if code, body := e.do(http.MethodPost, path, owner); code != http.StatusOK {
		t.Fatalf("owner: %d, want 200 (%v)", code, body)
	}
	var stored models.Booking
	if err := e.db.First(&stored, b.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.BookingStatusConfirmed {
		t.Errorf("booking status = %q, want confirmed", stored.Status)
	}
}

func TestAuthzAdminUsersByRole(t *testing.T) {
	e := newAuthzEnv(t)
	customer := testutil.CreateUser(t, e.db, "user")
	admin := testutil.CreateUser(t, e.db, "admin")

	if code, _ := e.do(http.MethodGet, "/api/admin/users", nil); code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d, want 401", code)
	}
	if code, _ := e.do(http.MethodGet, "/api/admin/users", customer); code != http.StatusForbidden {
		t.Errorf("user: %d, want 403", code)
	}
	if code, body := e.do(http.MethodGet, "/api/admin/users", admin); code != http.StatusOK {
		t.Errorf("admin: %d, want 200 (%v)", code, body)
	}
}
