package booking

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
)

const (
	testOwnerID    uint = 10
	testCustomerID uint = 20
	testAdminID    uint = 30
	testProviderID      = "prov-1"
	testServiceID       = "svc-1"
)

var (
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	clockRe = regexp.MustCompile(`^\d{2}:\d{2}$`)
)

func clock(t *testing.T, hhmm string) time.Time {
	t.Helper()
	v, err := time.Parse("15:04", hhmm)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// testEngine registers the booking routes behind a stub auth middleware that
// trusts the X-Test-User / X-Test-Role headers (absent header = anonymous).
func testEngine(h *Handlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	group.Use(func(c *gin.Context) {
		if id := c.GetHeader("X-Test-User"); id != "" {
			uid, _ := strconv.Atoi(id)
			c.Set(auth.UserContextKey, &auth.Claims{UserID: uint(uid), Role: c.GetHeader("X-Test-Role")})
		}
		c.Next()
	})
	h.Register(group)
	return engine
}

func doJSON(t *testing.T, engine *gin.Engine, method, path string, userID uint, role string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if userID != 0 {
		req.Header.Set("X-Test-User", strconv.Itoa(int(userID)))
		req.Header.Set("X-Test-Role", role)
	}
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	var body map[string]interface{}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return res, body
}

// assertNoKeys walks a decoded JSON value and fails if any object has one of
// the forbidden keys.
func assertNoKeys(t *testing.T, v interface{}, forbidden ...string) {
	t.Helper()
	switch val := v.(type) {
	case map[string]interface{}:
		for _, key := range forbidden {
			if _, ok := val[key]; ok {
				t.Errorf("response contains forbidden key %q: %v", key, val)
			}
		}
		for _, child := range val {
			assertNoKeys(t, child, forbidden...)
		}
	case []interface{}:
		for _, child := range val {
			assertNoKeys(t, child, forbidden...)
		}
	}
}

func seededHandlers(t *testing.T) (*Handlers, *fakeStore) {
	t.Helper()
	sid := testServiceID
	date := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	store := newFakeStore(
		models.Booking{ID: 1, UserID: testCustomerID, ProviderID: testProviderID, ServiceID: &sid, CustomerName: "Ana",
			BookingDate: date, StartTime: clock(t, "09:00"), EndTime: clock(t, "10:00"),
			Status: models.BookingStatusConfirmed, PaymentStatus: models.BookingPaymentPaid,
			AdminNotes: "internal only", CustomerNotes: "window seat",
			User: models.User{ID: testCustomerID, Email: "ana@example.test", Role: "user"}},
		models.Booking{ID: 2, UserID: testCustomerID, ProviderID: testProviderID, ServiceID: &sid, CustomerName: "Ana",
			BookingDate: date.AddDate(0, 0, 1), StartTime: clock(t, "11:00"), EndTime: clock(t, "12:00"),
			Status: models.BookingStatusPending, PaymentStatus: models.BookingPaymentUnpaid},
		models.Booking{ID: 3, UserID: 99, ProviderID: "other-provider", CustomerName: "Bo",
			BookingDate: date, StartTime: clock(t, "09:00"), EndTime: clock(t, "10:00"), Status: models.BookingStatusConfirmed},
	)
	store.providerNames = map[string]string{testProviderID: "Salon Satu"}
	store.serviceNames = map[string]string{testServiceID: "Haircut"}
	providers := &fakeProviderSvc{providers: map[string]*models.ServiceProvider{
		testProviderID: {ID: testProviderID, UserID: testOwnerID, BusinessName: "Salon Satu"},
	}}
	return NewHandlersWithService(&Service{repo: store}, providers), store
}

func TestListProviderBookingsAuthorization(t *testing.T) {
	h, _ := seededHandlers(t)
	engine := testEngine(h)
	path := "/api/providers/" + testProviderID + "/bookings"

	cases := []struct {
		name   string
		user   uint
		role   string
		path   string
		status int
	}{
		{"anonymous", 0, "", path, http.StatusUnauthorized},
		{"owner", testOwnerID, "provider", path, http.StatusOK},
		{"admin", testAdminID, "admin", path, http.StatusOK},
		{"other user", testCustomerID, "user", path, http.StatusForbidden},
		{"unknown provider", testOwnerID, "provider", "/api/providers/nope/bookings", http.StatusNotFound},
	}
	for _, tc := range cases {
		res, _ := doJSON(t, engine, http.MethodGet, tc.path, tc.user, tc.role)
		if res.Code != tc.status {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, res.Code, tc.status, res.Body.String())
		}
	}
}

func TestListProviderBookingsResponse(t *testing.T) {
	h, store := seededHandlers(t)
	engine := testEngine(h)

	res, body := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings", testOwnerID, "provider")
	if res.Code != http.StatusOK {
		t.Fatalf("status %d: %s", res.Code, res.Body.String())
	}
	list, ok := body["bookings"].([]interface{})
	if !ok || len(list) != 2 {
		t.Fatalf("bookings = %v", body["bookings"])
	}
	assertNoKeys(t, body, "email", "admin_notes", "role", "password", "user", "user_id")

	first := list[0].(map[string]interface{})
	if first["id"].(float64) != 2 {
		t.Errorf("first booking id = %v, want 2 (newest date first)", first["id"])
	}
	for _, item := range list {
		b := item.(map[string]interface{})
		if !dateRe.MatchString(b["booking_date"].(string)) {
			t.Errorf("booking_date %q", b["booking_date"])
		}
		for _, key := range []string{"start_time", "end_time"} {
			if !clockRe.MatchString(b[key].(string)) {
				t.Errorf("%s %q", key, b[key])
			}
		}
		if b["service_name"] != "Haircut" || b["customer_name"] != "Ana" {
			t.Errorf("names = %v / %v", b["service_name"], b["customer_name"])
		}
	}
	if list[1].(map[string]interface{})["start_time"] != "09:00" {
		t.Errorf("start_time = %v", list[1].(map[string]interface{})["start_time"])
	}

	t.Run("status filter", func(t *testing.T) {
		res, body := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings?status=pending", testOwnerID, "provider")
		if res.Code != http.StatusOK || len(body["bookings"].([]interface{})) != 1 || store.lastFilter != "pending" {
			t.Fatalf("status %d body %v filter %q", res.Code, body, store.lastFilter)
		}
	})
	t.Run("invalid status", func(t *testing.T) {
		res, _ := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings?status=bogus", testOwnerID, "provider")
		if res.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", res.Code)
		}
	})
	t.Run("negative offset treated as zero and limit clamped", func(t *testing.T) {
		res, _ := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings?offset=-5&limit=500", testOwnerID, "provider")
		if res.Code != http.StatusOK || store.lastListOffset != 0 || store.lastListLimit != 100 {
			t.Fatalf("status %d offset %d limit %d", res.Code, store.lastListOffset, store.lastListLimit)
		}
	})
	t.Run("empty list is an empty array", func(t *testing.T) {
		res, _ := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings?status=completed", testOwnerID, "provider")
		if res.Code != http.StatusOK || res.Body.String() != `{"bookings":[]}` {
			t.Fatalf("status %d body %s", res.Code, res.Body.String())
		}
	})
	t.Run("store error", func(t *testing.T) {
		store.listErr = errFake
		defer func() { store.listErr = nil }()
		res, _ := doJSON(t, engine, http.MethodGet, "/api/providers/"+testProviderID+"/bookings", testOwnerID, "provider")
		if res.Code != http.StatusInternalServerError {
			t.Fatalf("status %d, want 500", res.Code)
		}
	})
}

func TestCustomerBookingResponsesUseDTO(t *testing.T) {
	h, _ := seededHandlers(t)
	engine := testEngine(h)

	res, body := doJSON(t, engine, http.MethodGet, "/api/bookings/1", testCustomerID, "user")
	if res.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", res.Code, res.Body.String())
	}
	assertNoKeys(t, body, "admin_notes", "email", "user", "service_provider")
	b := body["booking"].(map[string]interface{})
	if b["provider_name"] != "Salon Satu" || b["service_name"] != "Haircut" || b["start_time"] != "09:00" || b["booking_date"] != "2030-01-07" {
		t.Fatalf("booking = %v", b)
	}

	res, _ = doJSON(t, engine, http.MethodGet, "/api/bookings/3", testCustomerID, "user")
	if res.Code != http.StatusNotFound {
		t.Fatalf("other user's booking status %d, want 404", res.Code)
	}

	res, body = doJSON(t, engine, http.MethodGet, "/api/bookings?offset=-1", testCustomerID, "user")
	if res.Code != http.StatusOK {
		t.Fatalf("list status %d", res.Code)
	}
	list := body["bookings"].([]interface{})
	if len(list) != 2 {
		t.Fatalf("list len %d", len(list))
	}
	assertNoKeys(t, body, "admin_notes", "email")
}

func TestServiceListProviderBookingsValidation(t *testing.T) {
	svc := &Service{repo: newFakeStore()}
	if _, err := svc.ListProviderBookings(context.Background(), "", "", 10, 0); err != ErrInvalidInput {
		t.Fatalf("empty provider err = %v", err)
	}
	for _, status := range []string{"", models.BookingStatusPending, models.BookingStatusConfirmed, models.BookingStatusCompleted, models.BookingStatusCancelled, models.BookingStatusPaymentFailed} {
		if _, err := svc.ListProviderBookings(context.Background(), "p", status, 10, 0); err != nil {
			t.Errorf("status %q err = %v", status, err)
		}
	}
	if _, err := svc.ListProviderBookings(context.Background(), "p", "PENDING", 10, 0); err != ErrInvalidInput {
		t.Errorf("uppercase status err = %v", err)
	}
}

func TestClampPage(t *testing.T) {
	cases := []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, 0, 20, 0},
		{-3, -3, 20, 0},
		{500, 10, 100, 10},
		{100, 0, 100, 0},
		{15, 5, 15, 5},
	}
	for _, tc := range cases {
		l, o := clampPage(tc.limit, tc.offset)
		if l != tc.wantLimit || o != tc.wantOffset {
			t.Errorf("clampPage(%d,%d) = %d,%d", tc.limit, tc.offset, l, o)
		}
	}
}

func doRescheduleJSON(t *testing.T, engine *gin.Engine, bookingID uint, userID uint, slot map[string]string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	payload, err := json.Marshal(map[string]interface{}{"slot": slot})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/bookings/"+strconv.Itoa(int(bookingID)), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User", strconv.Itoa(int(userID)))
	req.Header.Set("X-Test-Role", "user")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	var body map[string]interface{}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return res, body
}

func TestRescheduleHandlerStatusMapping(t *testing.T) {
	sid := testServiceID
	svcFor := func(t *testing.T, status string) (*gin.Engine, *fakeStore) {
		seed := models.Booking{ID: 1, UserID: testCustomerID, ProviderID: testProviderID, ServiceID: &sid,
			BookingDate: time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC), StartTime: clock(t, "09:00"), EndTime: clock(t, "10:00"),
			Status: status, PaymentStatus: models.BookingPaymentPaid}
		svc, d := newTestService(0, seed)
		providers := &fakeProviderSvc{providers: map[string]*models.ServiceProvider{testProviderID: {ID: testProviderID, UserID: testOwnerID}}}
		return testEngine(NewHandlersWithService(svc, providers)), d.store
	}
	slot := func(start, end string) map[string]string {
		return map[string]string{"date": "2030-02-01", "start_time": start, "end_time": end}
	}

	t.Run("ok", func(t *testing.T) {
		engine, store := svcFor(t, models.BookingStatusConfirmed)
		res, _ := doRescheduleJSON(t, engine, 1, testCustomerID, slot("14:00", "15:00"))
		if res.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", res.Code, res.Body)
		}
		if got := store.get(1); got.Status != models.BookingStatusConfirmed || got.StartTime.Format("15:04") != "14:00" {
			t.Fatalf("booking = %+v", got)
		}
	})
	for _, status := range []string{models.BookingStatusCancelled, models.BookingStatusCompleted} {
		t.Run(status+" is a conflict, not a server error", func(t *testing.T) {
			engine, _ := svcFor(t, status)
			res, _ := doRescheduleJSON(t, engine, 1, testCustomerID, slot("14:00", "15:00"))
			if res.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409: %s", res.Code, res.Body)
			}
		})
	}
	t.Run("wrong duration is a bad request", func(t *testing.T) {
		engine, _ := svcFor(t, models.BookingStatusConfirmed)
		res, _ := doRescheduleJSON(t, engine, 1, testCustomerID, slot("14:00", "14:30"))
		if res.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", res.Code, res.Body)
		}
	})
	t.Run("someone else's booking is not found", func(t *testing.T) {
		engine, _ := svcFor(t, models.BookingStatusConfirmed)
		res, _ := doRescheduleJSON(t, engine, 1, 99, slot("14:00", "15:00"))
		if res.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404: %s", res.Code, res.Body)
		}
	})
}
