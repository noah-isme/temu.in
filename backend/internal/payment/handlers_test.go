package payment

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
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

const (
	testServerKey = "test"
	testJWTSecret = "jwt-secret"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeStore is an in-memory paymentStore with the same transition rules as
// *Repository. writes counts every mutation so tests can assert "no writes".
type fakeStore struct {
	payments map[uint]*models.Payment
	events   map[string]uint
	nextID   uint
	writes   int
	clock    time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{payments: map[uint]*models.Payment{}, events: map[string]uint{}, nextID: 1, clock: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeStore) tick() time.Time {
	f.clock = f.clock.Add(time.Second)
	return f.clock
}

func (f *fakeStore) add(p *models.Payment) *models.Payment {
	p.ID = f.nextID
	f.nextID++
	p.CreatedAt, p.UpdatedAt = f.tick(), f.clock
	cp := *p
	f.payments[p.ID] = &cp
	return p
}

func (f *fakeStore) Create(p *models.Payment) error {
	f.writes++
	f.add(p)
	return nil
}

func (f *fakeStore) FindByOrderID(orderID string) (*models.Payment, error) {
	for _, p := range f.payments {
		if p.OrderID == orderID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindByClientSecret(cs string) (*models.Payment, error) {
	for _, p := range f.payments {
		if p.ClientSecret == cs {
			cp := *p
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) UpdateProviderStatus(id uint, status, transactionID string) (*models.Payment, error) {
	p, ok := f.payments[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if models.CanTransitionPaymentStatus(p.Status, status) &&
		!(p.Status == status && (transactionID == "" || transactionID == p.TransactionID)) {
		f.writes++
		p.Status = status
		if transactionID != "" {
			p.TransactionID = transactionID
		}
		p.UpdatedAt = f.tick()
	}
	cp := *p
	return &cp, nil
}

func (f *fakeStore) RecordWebhookEvent(key string, paymentID uint) (bool, error) {
	if _, ok := f.events[key]; ok {
		return false, nil
	}
	f.writes++
	f.events[key] = paymentID
	return true, nil
}

type finalizerCall struct {
	paymentID uint
	status    string
}

type countingFinalizer struct {
	calls []finalizerCall
	err   error
}

func (c *countingFinalizer) finalize(paymentID uint, status string) error {
	c.calls = append(c.calls, finalizerCall{paymentID, status})
	return c.err
}

type fakeCatalog struct{ services map[string]*models.Service }

func (f fakeCatalog) GetService(id string) (*models.Service, error) {
	if s, ok := f.services[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type fakeProvider struct {
	createErr error
	getErr    error
	status    string
	created   []CreateTransactionRequest
}

func (f *fakeProvider) CreateTransaction(_ context.Context, req CreateTransactionRequest) (*Transaction, error) {
	f.created = append(f.created, req)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &Transaction{OrderID: req.OrderID, Token: "snap-token", RedirectURL: "https://pay.example/redirect", Status: "pending"}, nil
}

func (f *fakeProvider) GetTransaction(_ context.Context, orderID string) (*Transaction, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &Transaction{OrderID: orderID, TransactionID: "tx-confirm", Status: f.status}, nil
}

func (f *fakeProvider) VerifyWebhook(WebhookNotification) bool { return true }

func newEngine(h *Handlers) *gin.Engine {
	engine := gin.New()
	api := engine.Group("/api")
	protected := api.Group("/")
	protected.Use(auth.Middleware(testJWTSecret))
	h.Register(protected)
	h.RegisterWebhook(api.Group("/"))
	return engine
}

func bearer(t *testing.T, userID uint) string {
	t.Helper()
	tok, err := auth.NewToken(testJWTSecret, userID, "user", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func do(engine *gin.Engine, method, path string, body []byte, authz string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func signedBody(t *testing.T, n WebhookNotification) []byte {
	t.Helper()
	digest := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + testServerKey))
	n.SignatureKey = hex.EncodeToString(digest[:])
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return out
}

type webhookEnv struct {
	store     *fakeStore
	finalizer *countingFinalizer
	engine    *gin.Engine
	payment   *models.Payment
}

func newWebhookEnv(t *testing.T) webhookEnv {
	t.Helper()
	store := newFakeStore()
	p := store.add(&models.Payment{UserID: 7, Provider: ProviderMidtrans, OrderID: "temu_order_1", IntentID: "temu_order_1", ClientSecret: "cs-1", Amount: 75000, Currency: "IDR", Status: models.PaymentStatusPending})
	fin := &countingFinalizer{}
	svc := NewService(nil, NewMidtransProvider(testServerKey, "", "http://127.0.0.1"))
	h := &Handlers{repo: store, service: svc, finalizeBooking: fin.finalize}
	return webhookEnv{store: store, finalizer: fin, engine: newEngine(h), payment: p}
}

func notificationFor(orderID, transactionStatus, fraud string) WebhookNotification {
	return WebhookNotification{OrderID: orderID, StatusCode: "200", GrossAmount: "75000.00", TransactionID: "tx-1", TransactionStatus: transactionStatus, FraudStatus: fraud}
}

func TestWebhookStatusMatrix(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		fraud         string
		start         string
		want          string
		wantFinalized bool
	}{
		{name: "settlement", status: "settlement", want: models.PaymentStatusSucceeded, wantFinalized: true},
		{name: "capture accept", status: "capture", fraud: "accept", want: models.PaymentStatusSucceeded, wantFinalized: true},
		{name: "capture challenge", status: "capture", fraud: "challenge", want: models.PaymentStatusPending, wantFinalized: true},
		{name: "capture deny", status: "capture", fraud: "deny", want: models.PaymentStatusFailed, wantFinalized: true},
		{name: "pending", status: "pending", want: models.PaymentStatusPending, wantFinalized: true},
		{name: "expire", status: "expire", want: models.PaymentStatusExpired, wantFinalized: true},
		{name: "deny", status: "deny", want: models.PaymentStatusFailed, wantFinalized: true},
		{name: "cancel", status: "cancel", want: models.PaymentStatusFailed, wantFinalized: true},
		{name: "failure", status: "failure", want: models.PaymentStatusFailed, wantFinalized: true},
		{name: "refund", status: "refund", start: models.PaymentStatusSucceeded, want: models.PaymentStatusRefunded},
		{name: "partial refund", status: "partial_refund", start: models.PaymentStatusSucceeded, want: models.PaymentStatusRefunded},
		{name: "chargeback", status: "chargeback", start: models.PaymentStatusSucceeded, want: models.PaymentStatusRefunded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newWebhookEnv(t)
			if tt.start != "" {
				env.store.payments[env.payment.ID].Status = tt.start
			}
			w := do(env.engine, http.MethodPost, "/api/payments/webhook", signedBody(t, notificationFor(env.payment.OrderID, tt.status, tt.fraud)), "")
			if w.Code != http.StatusOK {
				t.Fatalf("status %d body %s", w.Code, w.Body.String())
			}
			if got := decode(t, w)["status"]; got != "received" {
				t.Fatalf("response status = %v, want received", got)
			}
			if got := env.store.payments[env.payment.ID].Status; got != tt.want {
				t.Fatalf("payment status = %q, want %q", got, tt.want)
			}
			if tt.wantFinalized {
				if len(env.finalizer.calls) != 1 || env.finalizer.calls[0] != (finalizerCall{env.payment.ID, tt.want}) {
					t.Fatalf("finalizer calls = %+v, want one call with %q", env.finalizer.calls, tt.want)
				}
			} else if len(env.finalizer.calls) != 0 {
				t.Fatalf("finalizer must not run for %q, got %+v", tt.want, env.finalizer.calls)
			}
		})
	}
}

func TestWebhookReplayIsIdempotent(t *testing.T) {
	env := newWebhookEnv(t)
	body := signedBody(t, notificationFor(env.payment.OrderID, "settlement", ""))

	first := do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
	if first.Code != http.StatusOK || decode(t, first)["status"] != "received" {
		t.Fatalf("first delivery: %d %s", first.Code, first.Body.String())
	}
	before, _ := json.Marshal(env.store.payments[env.payment.ID])
	writesBefore := env.store.writes

	second := do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
	if second.Code != http.StatusOK || decode(t, second)["status"] != "duplicate" {
		t.Fatalf("replay: %d %s", second.Code, second.Body.String())
	}
	after, _ := json.Marshal(env.store.payments[env.payment.ID])
	if !bytes.Equal(before, after) {
		t.Fatalf("payment changed on replay:\nbefore %s\nafter  %s", before, after)
	}
	if env.store.writes != writesBefore {
		t.Fatalf("replay performed %d writes", env.store.writes-writesBefore)
	}
	if len(env.store.events) != 1 {
		t.Fatalf("webhook events = %d, want 1", len(env.store.events))
	}
	for _, call := range env.finalizer.calls {
		if call != env.finalizer.calls[0] {
			t.Fatalf("finalizer replay used different arguments: %+v", env.finalizer.calls)
		}
	}
}

func TestWebhookRejections(t *testing.T) {
	t.Run("invalid signature", func(t *testing.T) {
		env := newWebhookEnv(t)
		n := notificationFor(env.payment.OrderID, "settlement", "")
		n.SignatureKey = strings.Repeat("0", 128)
		body, _ := json.Marshal(n)
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", w.Code)
		}
		if env.store.writes != 0 || len(env.finalizer.calls) != 0 {
			t.Fatalf("writes=%d finalizer=%d, want none", env.store.writes, len(env.finalizer.calls))
		}
	})
	t.Run("gross amount mismatch", func(t *testing.T) {
		env := newWebhookEnv(t)
		n := notificationFor(env.payment.OrderID, "settlement", "")
		n.GrossAmount = "1.00"
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", signedBody(t, n), "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", w.Code)
		}
		if env.store.writes != 0 {
			t.Fatalf("writes=%d, want 0", env.store.writes)
		}
	})
	t.Run("unknown order", func(t *testing.T) {
		env := newWebhookEnv(t)
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", signedBody(t, notificationFor("missing", "settlement", "")), "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status %d, want 404", w.Code)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		env := newWebhookEnv(t)
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", []byte("{not json"), "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", w.Code)
		}
	})
	t.Run("body over 1 MiB", func(t *testing.T) {
		env := newWebhookEnv(t)
		body := append([]byte(`{"order_id":"`), bytes.Repeat([]byte("a"), 1<<20)...)
		body = append(body, []byte(`"}`)...)
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", w.Code)
		}
	})
	t.Run("service not configured", func(t *testing.T) {
		h := &Handlers{repo: newFakeStore()}
		w := do(newEngine(h), http.MethodPost, "/api/payments/webhook", []byte("{}"), "")
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want 503", w.Code)
		}
	})
	t.Run("finalizer error leaves event unrecorded", func(t *testing.T) {
		env := newWebhookEnv(t)
		env.finalizer.err = errors.New("db down")
		body := signedBody(t, notificationFor(env.payment.OrderID, "settlement", ""))
		w := do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status %d, want 500", w.Code)
		}
		if len(env.store.events) != 0 {
			t.Fatalf("webhook event recorded despite finalizer failure")
		}
		// Midtrans retries the same body; once the finalizer recovers it is processed.
		env.finalizer.err = nil
		w = do(env.engine, http.MethodPost, "/api/payments/webhook", body, "")
		if w.Code != http.StatusOK || decode(t, w)["status"] != "received" {
			t.Fatalf("retry: %d %s", w.Code, w.Body.String())
		}
	})
}

func newIntentEnv(provider Provider) (*fakeStore, *countingFinalizer, *gin.Engine) {
	store := newFakeStore()
	catalog := fakeCatalog{services: map[string]*models.Service{
		"svc-1":    {ID: "svc-1", ProviderID: "prov-1", Name: "Haircut", Price: 75000, IsActive: true},
		"inactive": {ID: "inactive", ProviderID: "prov-1", Name: "Old", Price: 1000, IsActive: false},
	}}
	fin := &countingFinalizer{}
	h := &Handlers{repo: store, service: NewService(nil, provider), serviceCatalog: catalog, finalizeBooking: fin.finalize}
	return store, fin, newEngine(h)
}

func TestCreateIntent(t *testing.T) {
	t.Run("created with server-side amount", func(t *testing.T) {
		prov := &fakeProvider{}
		store, _, engine := newIntentEnv(prov)
		w := do(engine, http.MethodPost, "/api/payments/create-intent", []byte(`{"providerId":"prov-1","serviceId":"svc-1","currency":"idr"}`), bearer(t, 7))
		if w.Code != http.StatusCreated {
			t.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if body["amount"] != float64(75000) || body["currency"] != "IDR" || body["client_secret"] != "snap-token" {
			t.Fatalf("unexpected body %v", body)
		}
		if len(prov.created) != 1 || prov.created[0].Amount != 75000 {
			t.Fatalf("provider requests %+v", prov.created)
		}
		if len(store.payments) != 1 {
			t.Fatalf("payments stored = %d", len(store.payments))
		}
	})
	cases := []struct {
		name     string
		provider Provider
		body     string
		want     int
	}{
		{"unknown service", &fakeProvider{}, `{"providerId":"prov-1","serviceId":"nope"}`, http.StatusNotFound},
		{"inactive service", &fakeProvider{}, `{"providerId":"prov-1","serviceId":"inactive"}`, http.StatusBadRequest},
		{"mismatched provider", &fakeProvider{}, `{"providerId":"prov-2","serviceId":"svc-1"}`, http.StatusBadRequest},
		{"non-IDR", &fakeProvider{}, `{"providerId":"prov-1","serviceId":"svc-1","currency":"USD"}`, http.StatusBadRequest},
		{"provider not configured", NewMidtransProvider("", "", "http://127.0.0.1"), `{"providerId":"prov-1","serviceId":"svc-1"}`, http.StatusServiceUnavailable},
		{"provider error", &fakeProvider{createErr: errors.New("boom")}, `{"providerId":"prov-1","serviceId":"svc-1"}`, http.StatusBadGateway},
		{"missing fields", &fakeProvider{}, `{}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, _, engine := newIntentEnv(tc.provider)
			w := do(engine, http.MethodPost, "/api/payments/create-intent", []byte(tc.body), bearer(t, 7))
			if w.Code != tc.want {
				t.Fatalf("status %d, want %d (body %s)", w.Code, tc.want, w.Body.String())
			}
			if len(store.payments) != 0 {
				t.Fatalf("payment stored on failure")
			}
		})
	}
	t.Run("payments disabled", func(t *testing.T) {
		engine := newEngine(&Handlers{repo: newFakeStore()})
		w := do(engine, http.MethodPost, "/api/payments/create-intent", []byte(`{"providerId":"prov-1","serviceId":"svc-1"}`), bearer(t, 7))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want 503", w.Code)
		}
	})
	t.Run("unauthenticated", func(t *testing.T) {
		_, _, engine := newIntentEnv(&fakeProvider{})
		w := do(engine, http.MethodPost, "/api/payments/create-intent", []byte(`{"providerId":"prov-1","serviceId":"svc-1"}`), "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", w.Code)
		}
	})
}

func TestConfirm(t *testing.T) {
	seed := func(store *fakeStore) *models.Payment {
		return store.add(&models.Payment{UserID: 7, Provider: ProviderMidtrans, OrderID: "temu_c", IntentID: "temu_c", ClientSecret: "cs-c", Amount: 75000, Currency: "IDR", Status: models.PaymentStatusPending})
	}
	t.Run("ok by order id", func(t *testing.T) {
		store, fin, engine := newIntentEnv(&fakeProvider{status: models.PaymentStatusSucceeded})
		p := seed(store)
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"order_id":"temu_c"}`), bearer(t, 7))
		if w.Code != http.StatusOK {
			t.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if body["status"] != models.PaymentStatusSucceeded || body["transactionId"] != "tx-confirm" || body["order_id"] != "temu_c" {
			t.Fatalf("unexpected body %v", body)
		}
		if len(fin.calls) != 1 || fin.calls[0] != (finalizerCall{p.ID, models.PaymentStatusSucceeded}) {
			t.Fatalf("finalizer calls %+v", fin.calls)
		}
	})
	t.Run("ok by client secret", func(t *testing.T) {
		store, _, engine := newIntentEnv(&fakeProvider{status: models.PaymentStatusPending})
		seed(store)
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"client_secret":"cs-c"}`), bearer(t, 7))
		if w.Code != http.StatusOK {
			t.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
	})
	t.Run("unknown payment", func(t *testing.T) {
		_, _, engine := newIntentEnv(&fakeProvider{})
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"order_id":"nope"}`), bearer(t, 7))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status %d, want 404", w.Code)
		}
	})
	t.Run("missing identifiers", func(t *testing.T) {
		_, _, engine := newIntentEnv(&fakeProvider{})
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{}`), bearer(t, 7))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", w.Code)
		}
	})
	t.Run("other user's payment", func(t *testing.T) {
		store, fin, engine := newIntentEnv(&fakeProvider{status: models.PaymentStatusSucceeded})
		seed(store)
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"order_id":"temu_c"}`), bearer(t, 8))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status %d, want 403", w.Code)
		}
		if len(fin.calls) != 0 {
			t.Fatalf("finalizer ran for forbidden request")
		}
	})
	t.Run("provider error", func(t *testing.T) {
		store, _, engine := newIntentEnv(&fakeProvider{getErr: errors.New("boom")})
		seed(store)
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"order_id":"temu_c"}`), bearer(t, 7))
		if w.Code != http.StatusBadGateway {
			t.Fatalf("status %d, want 502", w.Code)
		}
	})
	t.Run("unauthenticated", func(t *testing.T) {
		_, _, engine := newIntentEnv(&fakeProvider{})
		w := do(engine, http.MethodPost, "/api/payments/confirm", []byte(`{"order_id":"temu_c"}`), "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", w.Code)
		}
	})
}

func TestServiceAmount(t *testing.T) {
	if _, err := serviceAmount(-1); err == nil {
		t.Fatal("negative price accepted")
	}
	if _, err := serviceAmount(0.2); err == nil {
		t.Fatal("price rounding to zero accepted")
	}
	if got, err := serviceAmount(74999.6); err != nil || got != 75000 {
		t.Fatalf("got %d, %v", got, err)
	}
}
