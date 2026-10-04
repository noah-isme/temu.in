package payment_test

// External test package: it wires the real booking repository (booking
// imports payment, so an in-package test could not).

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/booking"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/notification"
	"temu.in/backend/internal/payment"
	"temu.in/backend/internal/service"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

const itServerKey = "test"

type paymentIT struct {
	db          *gorm.DB
	payments    *payment.Repository
	bookings    *booking.Repository
	engine      *gin.Engine
	payment     *models.Payment
	bookingID   uint
	webhookPath string
}

var itOrderSeq int

// newPaymentIT wires the webhook exactly like router.registerPayments: real
// payment, booking and notification repositories plus the production
// finalizer. It seeds a pending paid booking linked to a pending payment.
func newPaymentIT(t *testing.T) paymentIT {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	customer := testutil.CreateUser(t, db, "user")
	provider := testutil.CreateProvider(t, db, owner.ID)
	svc := testutil.CreateService(t, db, provider.ID, 60, 75000)

	paymentRepo := payment.NewRepository(db)
	bookingRepo := booking.NewRepository(db)
	notifier := notification.NewService(notification.NewRepository(db))
	finalizer := payment.NewBookingFinalizer(bookingRepo, paymentRepo, notifier, nil)
	handlers := payment.NewHandlersWithDependencies(paymentRepo,
		payment.NewService(paymentRepo, payment.NewMidtransProvider(itServerKey, "", "http://127.0.0.1")),
		service.NewService(service.NewRepository(db)), finalizer)
	engine := gin.New()
	handlers.RegisterWebhook(engine.Group("/api"))

	itOrderSeq++
	orderID := fmt.Sprintf("temu_it_%d", itOrderSeq)
	p := &models.Payment{UserID: customer.ID, Provider: payment.ProviderMidtrans, OrderID: orderID, IntentID: orderID, ClientSecret: "cs-" + orderID, Amount: 75000, Currency: "IDR", Status: models.PaymentStatusPending}
	b := &models.Booking{
		UserID:       customer.ID,
		ProviderID:   provider.ID,
		ServiceID:    &svc.ID,
		CustomerName: "Customer",
		BookingDate:  testutil.MustDate(t, "2030-01-07"),
		StartTime:    testutil.MustClock(t, "10:00"),
		EndTime:      testutil.MustClock(t, "11:00"),
	}
	if err := bookingRepo.CreateWithPayment(b, p); err != nil {
		t.Fatalf("create booking with payment: %v", err)
	}
	return paymentIT{db: db, payments: paymentRepo, bookings: bookingRepo, engine: engine, payment: p, bookingID: b.ID, webhookPath: "/api/payments/webhook"}
}

func (f paymentIT) settlementBody(t *testing.T) []byte {
	t.Helper()
	n := payment.WebhookNotification{OrderID: f.payment.OrderID, StatusCode: "200", GrossAmount: "75000.00", TransactionID: "tx-it", TransactionStatus: "settlement"}
	digest := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + itServerKey))
	n.SignatureKey = hex.EncodeToString(digest[:])
	body, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func (f paymentIT) post(t *testing.T, body []byte) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, f.webhookPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.engine.ServeHTTP(w, req)
	var out struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Error != "" {
		return w.Code, out.Error
	}
	return w.Code, out.Status
}

func rows(t *testing.T, db *gorm.DB, query string, args ...any) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := db.Raw(query, args...).Scan(&out).Error; err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return out
}

func count(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestWebhookReplayIdempotenceIntegration(t *testing.T) {
	f := newPaymentIT(t)
	body := f.settlementBody(t)

	if code, status := f.post(t, body); code != http.StatusOK || status != "received" {
		t.Fatalf("first delivery: %d %s", code, status)
	}
	paymentsBefore := rows(t, f.db, `SELECT * FROM payments ORDER BY id`)
	bookingsBefore := rows(t, f.db, `SELECT * FROM bookings ORDER BY id`)
	if paymentsBefore[0]["status"] != models.PaymentStatusSucceeded {
		t.Fatalf("payment status = %v", paymentsBefore[0]["status"])
	}
	if bookingsBefore[0]["status"] != models.BookingStatusConfirmed || bookingsBefore[0]["payment_status"] != models.BookingPaymentPaid {
		t.Fatalf("booking = %v / %v", bookingsBefore[0]["status"], bookingsBefore[0]["payment_status"])
	}
	notificationsBefore := count(t, f.db, `SELECT count(*) FROM notifications`)
	if notificationsBefore != 3 {
		t.Fatalf("notifications after settlement = %d, want 3 (customer+provider confirmation, receipt)", notificationsBefore)
	}

	if code, status := f.post(t, body); code != http.StatusOK || status != "duplicate" {
		t.Fatalf("replay: %d %s", code, status)
	}
	if after := rows(t, f.db, `SELECT * FROM payments ORDER BY id`); !reflect.DeepEqual(paymentsBefore, after) {
		t.Fatalf("payments changed on replay:\nbefore %v\nafter  %v", paymentsBefore, after)
	}
	if after := rows(t, f.db, `SELECT * FROM bookings ORDER BY id`); !reflect.DeepEqual(bookingsBefore, after) {
		t.Fatalf("bookings changed on replay:\nbefore %v\nafter  %v", bookingsBefore, after)
	}
	if n := count(t, f.db, `SELECT count(*) FROM payment_webhook_events`); n != 1 {
		t.Fatalf("payment_webhook_events = %d, want 1", n)
	}
	if n := count(t, f.db, `SELECT count(*) FROM notifications`); n != notificationsBefore {
		t.Fatalf("notifications = %d after replay, want %d", n, notificationsBefore)
	}
	if n := count(t, f.db, `SELECT count(*) FROM (SELECT deduplication_key FROM notifications GROUP BY deduplication_key HAVING count(*) > 1) d`); n != 0 {
		t.Fatalf("%d deduplication keys have more than one row", n)
	}
}

func TestSettlementOnCancelledBookingIntegration(t *testing.T) {
	f := newPaymentIT(t)
	if err := f.bookings.UpdateStatus(f.bookingID, models.BookingStatusCancelled, "changed my mind"); err != nil {
		t.Fatalf("cancel booking: %v", err)
	}

	if code, status := f.post(t, f.settlementBody(t)); code != http.StatusOK || status != "received" {
		t.Fatalf("settlement: %d %s", code, status)
	}
	p, err := f.payments.FindByID(f.payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != models.PaymentStatusSucceeded {
		t.Fatalf("payment status = %q, want succeeded", p.Status)
	}
	b, err := f.bookings.FindByID(f.bookingID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != models.BookingStatusCancelled {
		t.Fatalf("booking status = %q, want cancelled", b.Status)
	}
	if n := count(t, f.db, `SELECT count(*) FROM notifications WHERE type IN (?, ?, ?)`,
		models.NotificationTypeBookingConfirmationCustomer, models.NotificationTypeBookingConfirmationProvider, models.NotificationTypePaymentReceipt); n != 0 {
		t.Fatalf("confirmation/receipt notifications = %d, want 0", n)
	}
}

func TestUpdateProviderStatusNeverMovesBackwardsIntegration(t *testing.T) {
	f := newPaymentIT(t)
	if _, err := f.payments.UpdateProviderStatus(f.payment.ID, models.PaymentStatusSucceeded, "tx-1"); err != nil {
		t.Fatal(err)
	}
	before := rows(t, f.db, `SELECT * FROM payments WHERE id = ?`, f.payment.ID)

	got, err := f.payments.UpdateProviderStatus(f.payment.ID, models.PaymentStatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.PaymentStatusSucceeded {
		t.Fatalf("succeeded -> pending applied: %q", got.Status)
	}
	// Same state again is a true no-op (updated_at untouched).
	if _, err := f.payments.UpdateProviderStatus(f.payment.ID, models.PaymentStatusSucceeded, "tx-1"); err != nil {
		t.Fatal(err)
	}
	if after := rows(t, f.db, `SELECT * FROM payments WHERE id = ?`, f.payment.ID); !reflect.DeepEqual(before, after) {
		t.Fatalf("payment row changed:\nbefore %v\nafter  %v", before, after)
	}

	got, err = f.payments.UpdateProviderStatus(f.payment.ID, models.PaymentStatusRefunded, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.PaymentStatusRefunded {
		t.Fatalf("succeeded -> refunded not applied: %q", got.Status)
	}
	if stored, _ := f.payments.FindByID(f.payment.ID); stored.Status != models.PaymentStatusRefunded || stored.TransactionID != "tx-1" {
		t.Fatalf("stored payment = %+v", stored)
	}
}

func TestRepositoryLookupsIntegration(t *testing.T) {
	f := newPaymentIT(t)
	if p, err := f.payments.FindByOrderID(f.payment.OrderID); err != nil || p.ID != f.payment.ID {
		t.Fatalf("FindByOrderID: %v %v", p, err)
	}
	if p, err := f.payments.FindByClientSecret(f.payment.ClientSecret); err != nil || p.ID != f.payment.ID {
		t.Fatalf("FindByClientSecret: %v %v", p, err)
	}
	if p, err := f.payments.FindByIntent(f.payment.IntentID); err != nil || p.ID != f.payment.ID {
		t.Fatalf("FindByIntent: %v %v", p, err)
	}
	if _, err := f.payments.FindByOrderID("missing"); err == nil {
		t.Fatal("expected error for unknown order")
	}
	claimed, err := f.payments.RecordWebhookEvent("key-1", f.payment.ID)
	if err != nil || !claimed {
		t.Fatalf("first claim: %v %v", claimed, err)
	}
	claimed, err = f.payments.RecordWebhookEvent("key-1", f.payment.ID)
	if err != nil || claimed {
		t.Fatalf("duplicate claim: %v %v", claimed, err)
	}
	f.payment.Currency = "IDR"
	f.payment.Status = models.PaymentStatusPending
	if err := f.payments.Update(f.payment); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if f.payments.DB() == nil {
		t.Fatal("DB() returned nil")
	}
}
