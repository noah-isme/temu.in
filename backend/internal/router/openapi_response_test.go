package router

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/booking"
	"temu.in/backend/internal/config"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
	"temu.in/backend/openapi"
)

// Decision recorded (plan item 5): the contract is OpenAPI 3.0.3 validated
// with kin-openapi rather than 3.1 + libopenapi-validator, for kin-openapi's
// maturity and stable openapi3filter.ValidateResponse API.

const responseTestHost = "http://api.test"

type specValidator struct {
	router routers.Router
}

// newSpecValidator loads and validates the spec document itself, then builds
// a router. The relative `servers: [{url: /}]` entry is replaced with an
// absolute test host so gorillamux can match httptest requests.
func newSpecValidator(t *testing.T, spec []byte) *specValidator {
	t.Helper()
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(spec)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("spec is not a valid OpenAPI document: %v", err)
	}
	doc.Servers = openapi3.Servers{{URL: responseTestHost}}
	r, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return &specValidator{router: r}
}

// validate checks one captured response against the operation matching req.
func (v *specValidator) validate(req *http.Request, status int, header http.Header, body []byte, skipBody bool) error {
	routeReq := req.Clone(context.Background())
	routeReq.URL.Scheme, routeReq.URL.Host, routeReq.Host = "http", "api.test", "api.test"
	route, params, err := v.router.FindRoute(routeReq)
	if err != nil {
		return fmt.Errorf("%s %s: no matching operation in spec: %w", req.Method, req.URL.Path, err)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: routeReq, PathParams: params, Route: route},
		Status:                 status,
		Header:                 header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, ExcludeResponseBody: skipBody, MultiError: true},
	}
	input.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		return fmt.Errorf("%s %s -> %d does not match the spec: %w\nbody: %s", req.Method, req.URL.Path, status, err, body)
	}
	return nil
}

type capture struct {
	req    *http.Request
	status int
	header http.Header
	body   []byte
}

// call issues a request through the engine and returns the captured response.
func call(t *testing.T, engine *gin.Engine, method, path, bearer string, payload any) capture {
	t.Helper()
	var raw []byte
	switch p := payload.(type) {
	case nil:
	case string:
		raw = []byte(p)
	case []byte:
		raw = p
	default:
		var err error
		if raw, err = json.Marshal(p); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if raw != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	return capture{req: req, status: w.Code, header: w.Header(), body: body}
}

func (v *specValidator) mustMatch(t *testing.T, c capture, wantStatus int) {
	t.Helper()
	if c.status != wantStatus {
		t.Errorf("%s %s: status %d, want %d (body %s)", c.req.Method, c.req.URL.Path, c.status, wantStatus, c.body)
		return
	}
	if err := v.validate(c.req, c.status, c.header, c.body, false); err != nil {
		t.Error(err)
	}
}

func TestOpenAPIResponsesUnitMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: "test"}, nil, nil).RegisterAll()
	v := newSpecValidator(t, openapi.Spec)

	v.mustMatch(t, call(t, engine, http.MethodGet, "/api/health/live", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, engine, http.MethodGet, "/health/live", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, engine, http.MethodGet, "/api/health/ready", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, engine, http.MethodGet, "/health/ready", "", nil), http.StatusOK)

	spec := call(t, engine, http.MethodGet, "/api/openapi.yaml", "", nil)
	if spec.status != http.StatusOK || spec.header.Get("Content-Type") != "application/yaml; charset=utf-8" {
		t.Errorf("openapi.yaml: %d %q", spec.status, spec.header.Get("Content-Type"))
	}
	if err := v.validate(spec.req, spec.status, spec.header, spec.body, true); err != nil {
		t.Error(err)
	}

	validation := call(t, engine, http.MethodPost, "/api/auth/login", "", `{}`)
	v.mustMatch(t, validation, http.StatusBadRequest)
	if !strings.Contains(string(validation.body), `"details"`) {
		t.Errorf("login with missing fields should be a ValidationErrorResponse: %s", validation.body)
	}
	v.mustMatch(t, call(t, engine, http.MethodPost, "/api/auth/login", "", `{not json`), http.StatusBadRequest)

	for _, path := range []string{"/api/me", "/api/bookings", "/api/providers/me", "/api/reviews/me", "/api/favorites/", "/api/admin/users"} {
		v.mustMatch(t, call(t, engine, http.MethodGet, path, "", nil), http.StatusUnauthorized)
	}
	v.mustMatch(t, call(t, engine, http.MethodPost, "/api/payments/create-intent", "", `{}`), http.StatusUnauthorized)
}

// customerBookingsBody is a well-formed GET /api/bookings body.
const customerBookingsBody = `{"bookings":[{"id":1,"provider_id":"7d3c2f6e-1d2a-4c55-9a8e-3b8f1f0a9b11","booking_date":"%s","start_time":"09:00","end_time":"10:00","status":"confirmed","payment_status":"not_required"}]}`

// TestOpenAPIResponseFormatValidationIsActive proves the response validator
// enforces `format: date` (and so would catch a DTO regression), without a DB.
func TestOpenAPIResponseFormatValidationIsActive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	header := http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}
	v := newSpecValidator(t, openapi.Spec)

	if err := v.validate(req, http.StatusOK, header, []byte(fmt.Sprintf(customerBookingsBody, "2030-01-07")), false); err != nil {
		t.Fatalf("well-formed body rejected: %v", err)
	}
	err := v.validate(req, http.StatusOK, header, []byte(fmt.Sprintf(customerBookingsBody, "2030-01-07T00:00:00Z")), false)
	if err == nil || !strings.Contains(err.Error(), "booking_date") {
		t.Fatalf("date-time value accepted for a format: date field (err %v)", err)
	}

	// Deliberate spec mismatch: Date declared as date-time must reject the
	// real "YYYY-MM-DD" output.
	mutated := mutateDateFormat(t)
	err = newSpecValidator(t, mutated).validate(req, http.StatusOK, header, []byte(fmt.Sprintf(customerBookingsBody, "2030-01-07")), false)
	if err == nil || !strings.Contains(err.Error(), "booking_date") {
		t.Fatalf("mutated spec (Date: date-time) accepted a date (err %v)", err)
	}
}

// mutateDateFormat returns the spec with the shared Date schema switched to
// `format: date-time`.
func mutateDateFormat(t *testing.T) []byte {
	t.Helper()
	const original = "    Date:\n      type: string\n      format: date\n      example: '2030-01-07'\n"
	if !bytes.Contains(openapi.Spec, []byte(original)) {
		t.Fatal("Date schema not found in spec; update mutateDateFormat")
	}
	return bytes.Replace(openapi.Spec, []byte(original), []byte("    Date:\n      type: string\n      format: date-time\n"), 1)
}

const responseServerKey = "response-test-server-key"

type responseEnv struct {
	db       *gorm.DB
	engine   *gin.Engine
	owner    *models.User
	customer *models.User
	admin    *models.User
	provider *models.ServiceProvider
	service  *models.Service
}

func newResponseEnv(t *testing.T) responseEnv {
	t.Helper()
	db := testutil.DB(t)
	cache, _ := testutil.Redis(t)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: authzSecret, MidtransServerKey: responseServerKey, AccessTokenTTL: time.Hour}, db, cache).RegisterAll()

	owner := testutil.CreateUser(t, db, "provider")
	customer := testutil.CreateUser(t, db, "user")
	admin := testutil.CreateUser(t, db, "admin")
	p := testutil.CreateProvider(t, db, owner.ID)
	svc := testutil.CreateService(t, db, p.ID, 60, 0)
	testutil.CreateSchedule(t, db, p.ID, 1, "09:00", "17:00") // Mondays
	return responseEnv{db: db, engine: engine, owner: owner, customer: customer, admin: admin, provider: p, service: svc}
}

func (e responseEnv) token(t *testing.T, u *models.User) string {
	t.Helper()
	tok, err := auth.NewToken(authzSecret, u.ID, u.Role, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestOpenAPIResponsesIntegration(t *testing.T) {
	e := newResponseEnv(t)
	v := newSpecValidator(t, openapi.Spec)
	owner, customer, admin := e.token(t, e.owner), e.token(t, e.customer), e.token(t, e.admin)
	pid := e.provider.ID

	// Reviews need a completed booking.
	done := testutil.CreateBooking(t, e.db, e.customer.ID, pid, &e.service.ID, "2030-01-14", "09:00", "10:00", models.BookingStatusCompleted)
	if err := e.db.Omit("Booking", "Provider", "User").Create(&models.Review{BookingID: done.ID, ProviderID: pid, UserID: e.customer.ID, Rating: 5, Comment: "great"}).Error; err != nil {
		t.Fatal(err)
	}
	exc := map[string]any{"date": "2030-01-21", "is_available": false, "reason": "holiday"}

	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid, "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/00000000-0000-0000-0000-000000000000", "", nil), http.StatusNotFound)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/me", owner, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/availability?date=2030-01-07&serviceId="+e.service.ID, "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/availability?providerId="+pid+"&date=2030-01-07&serviceId="+e.service.ID, "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/availability?date=2030-01-07", "", nil), http.StatusBadRequest)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/schedules", owner, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/providers/"+pid+"/schedules", owner, map[string]any{"day_of_week": 2, "start_time": "09:00", "end_time": "12:00", "is_available": true}), http.StatusCreated)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/providers/"+pid+"/schedules", customer, map[string]any{"day_of_week": 2, "start_time": "09:00", "end_time": "12:00"}), http.StatusForbidden)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/providers/"+pid+"/exceptions", owner, exc), http.StatusCreated)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/exceptions?start_date=2030-01-01&end_date=2030-02-01", owner, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/services", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/services", "", nil), http.StatusOK)

	slot := map[string]string{"date": "2030-01-07", "start_time": "09:00", "end_time": "10:00"}
	created := call(t, e.engine, http.MethodPost, "/api/bookings", customer, map[string]any{"providerId": pid, "serviceId": e.service.ID, "customerName": "Ann", "slot": slot})
	v.mustMatch(t, created, http.StatusCreated)
	if strings.Contains(string(created.body), `"payment"`) {
		t.Errorf("free booking must not carry a payment: %s", created.body)
	}
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/bookings", customer, map[string]any{"providerId": pid, "serviceId": e.service.ID, "customerName": "Ann", "slot": slot}), http.StatusConflict)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/bookings", customer, nil), http.StatusOK)
	var createdBody struct {
		Booking struct {
			ID uint `json:"id"`
		} `json:"booking"`
	}
	_ = json.Unmarshal(created.body, &createdBody)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, fmt.Sprintf("/api/bookings/%d", createdBody.Booking.ID), customer, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/bookings", owner, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/stats", owner, nil), http.StatusOK)

	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/reviews", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/providers/"+pid+"/reviews/stats", "", nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/reviews/me", customer, nil), http.StatusOK)

	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/me", customer, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/admin/users", admin, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/admin/audit", admin, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/admin/stats", admin, nil), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/favorites/toggle", customer, map[string]string{"serviceId": e.service.ID}), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodGet, "/api/favorites/", customer, nil), http.StatusOK)

	ready := call(t, e.engine, http.MethodGet, "/api/health/ready", "", nil)
	v.mustMatch(t, ready, http.StatusOK)
	if !strings.Contains(string(ready.body), `"postgres"`) || !strings.Contains(string(ready.body), `"latency_ms"`) {
		t.Errorf("readiness should report checks with latency: %s", ready.body)
	}

	// Payment webhook: a signed settlement for a pending paid booking, a
	// replay, and a bad signature.
	order := seedPaidBooking(t, e)
	settlement := signedWebhook(t, order, "settlement")
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/payments/webhook", "", settlement), http.StatusOK)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/payments/webhook", "", settlement), http.StatusOK)
	var forged map[string]string
	_ = json.Unmarshal(settlement, &forged)
	forged["signature_key"] = strings.Repeat("0", 128)
	v.mustMatch(t, call(t, e.engine, http.MethodPost, "/api/payments/webhook", "", forged), http.StatusUnauthorized)
}

// TestOpenAPIResponseMismatchDetectedIntegration repeats the deliberate
// mismatch against a real booking response.
func TestOpenAPIResponseMismatchDetectedIntegration(t *testing.T) {
	e := newResponseEnv(t)
	testutil.CreateBooking(t, e.db, e.customer.ID, e.provider.ID, &e.service.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusConfirmed)
	list := call(t, e.engine, http.MethodGet, "/api/bookings", e.token(t, e.customer), nil)
	newSpecValidator(t, openapi.Spec).mustMatch(t, list, http.StatusOK)

	err := newSpecValidator(t, mutateDateFormat(t)).validate(list.req, list.status, list.header, list.body, false)
	if err == nil || !strings.Contains(err.Error(), "booking_date") {
		t.Fatalf("real booking response accepted by a spec declaring booking_date as date-time (err %v)", err)
	}
}

func seedPaidBooking(t *testing.T, e responseEnv) string {
	t.Helper()
	order := fmt.Sprintf("temu_resp_%d", time.Now().UnixNano())
	p := &models.Payment{UserID: e.customer.ID, Provider: "midtrans", OrderID: order, IntentID: order, ClientSecret: "cs-" + order, Amount: 75000, Currency: "IDR", Status: models.PaymentStatusPending}
	b := &models.Booking{UserID: e.customer.ID, ProviderID: e.provider.ID, ServiceID: &e.service.ID, CustomerName: "Paid",
		BookingDate: testutil.MustDate(t, "2030-01-07"), StartTime: testutil.MustClock(t, "13:00"), EndTime: testutil.MustClock(t, "14:00")}
	if err := booking.NewRepository(e.db).CreateWithPayment(b, p); err != nil {
		t.Fatal(err)
	}
	return order
}

func signedWebhook(t *testing.T, orderID, status string) []byte {
	t.Helper()
	n := map[string]string{"order_id": orderID, "status_code": "200", "gross_amount": "75000.00", "transaction_id": "tx-" + orderID, "transaction_status": status}
	digest := sha512.Sum512([]byte(n["order_id"] + n["status_code"] + n["gross_amount"] + responseServerKey))
	n["signature_key"] = hex.EncodeToString(digest[:])
	raw, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
