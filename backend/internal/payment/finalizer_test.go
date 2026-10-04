package payment

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"temu.in/backend/internal/models"
)

type fakeFinalizerBookings struct {
	resultStatus string
	setErr       error
	findErr      error
	setCalls     []finalizerCall
}

func (f *fakeFinalizerBookings) SetStatusByPayment(paymentID uint, status string) (string, error) {
	f.setCalls = append(f.setCalls, finalizerCall{paymentID, status})
	return f.resultStatus, f.setErr
}

func (f *fakeFinalizerBookings) FindByPaymentID(paymentID uint) (*models.Booking, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return &models.Booking{ID: 11, ProviderID: "prov-1", PaymentID: &paymentID}, nil
}

type fakeFinalizerPayments struct{}

func (fakeFinalizerPayments) FindByID(id uint) (*models.Payment, error) {
	return &models.Payment{ID: id, Amount: 75000, Currency: "IDR"}, nil
}

type recordingNotifier struct {
	confirmations int
	receipts      int
}

func (r *recordingNotifier) SendBookingConfirmation(context.Context, *models.Booking, *models.User, *models.ServiceProvider) error {
	r.confirmations++
	return nil
}
func (r *recordingNotifier) SendBookingCancellation(context.Context, *models.Booking, *models.User, *models.ServiceProvider) error {
	return nil
}
func (r *recordingNotifier) SendBookingReminder(context.Context, *models.Booking, *models.User) error {
	return nil
}
func (r *recordingNotifier) SendPasswordReset(context.Context, *models.User, string) error {
	return nil
}
func (r *recordingNotifier) SendEmailVerification(context.Context, *models.User, string) error {
	return nil
}
func (r *recordingNotifier) SendPaymentReceipt(context.Context, *models.Booking, *models.User, *models.Payment) error {
	r.receipts++
	return nil
}

func TestBookingFinalizerGating(t *testing.T) {
	tests := []struct {
		name          string
		paymentStatus string
		bookingStatus string
		wantNotified  bool
	}{
		{"succeeded and confirmed notifies", models.PaymentStatusSucceeded, models.BookingStatusConfirmed, true},
		{"succeeded on cancelled booking is silent", models.PaymentStatusSucceeded, models.BookingStatusCancelled, false},
		{"succeeded without linked booking is silent", models.PaymentStatusSucceeded, "", false},
		{"failed payment is silent", models.PaymentStatusFailed, models.BookingStatusPaymentFailed, false},
		{"pending payment is silent", models.PaymentStatusPending, models.BookingStatusPending, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookings := &fakeFinalizerBookings{resultStatus: tt.bookingStatus}
			notifier := &recordingNotifier{}
			if err := NewBookingFinalizer(bookings, fakeFinalizerPayments{}, notifier, nil)(5, tt.paymentStatus); err != nil {
				t.Fatalf("finalize: %v", err)
			}
			if len(bookings.setCalls) != 1 || bookings.setCalls[0] != (finalizerCall{5, tt.paymentStatus}) {
				t.Fatalf("SetStatusByPayment calls %+v", bookings.setCalls)
			}
			notified := notifier.confirmations == 1 && notifier.receipts == 1
			silent := notifier.confirmations == 0 && notifier.receipts == 0
			if tt.wantNotified && !notified || !tt.wantNotified && !silent {
				t.Fatalf("confirmations=%d receipts=%d, wantNotified=%v", notifier.confirmations, notifier.receipts, tt.wantNotified)
			}
		})
	}
}

func TestBookingFinalizerErrors(t *testing.T) {
	t.Run("status update error is returned", func(t *testing.T) {
		bookings := &fakeFinalizerBookings{setErr: errors.New("db down")}
		if err := NewBookingFinalizer(bookings, fakeFinalizerPayments{}, &recordingNotifier{}, nil)(5, models.PaymentStatusSucceeded); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing booking is not an error", func(t *testing.T) {
		bookings := &fakeFinalizerBookings{resultStatus: models.BookingStatusConfirmed, findErr: gorm.ErrRecordNotFound}
		if err := NewBookingFinalizer(bookings, fakeFinalizerPayments{}, &recordingNotifier{}, nil)(5, models.PaymentStatusSucceeded); err != nil {
			t.Fatalf("finalize: %v", err)
		}
	})
	t.Run("nil notifier", func(t *testing.T) {
		bookings := &fakeFinalizerBookings{resultStatus: models.BookingStatusConfirmed}
		if err := NewBookingFinalizer(bookings, fakeFinalizerPayments{}, nil, nil)(5, models.PaymentStatusSucceeded); err != nil {
			t.Fatalf("finalize: %v", err)
		}
	})
}

type recordingSlots struct{ providers []string }

func (r *recordingSlots) InvalidateProviderSlots(_ context.Context, providerID string) {
	r.providers = append(r.providers, providerID)
}

func TestBookingFinalizerSlotInvalidation(t *testing.T) {
	tests := []struct {
		name           string
		paymentStatus  string
		bookingStatus  string
		wantInvalidate bool
	}{
		{"payment failed frees slot", models.PaymentStatusFailed, models.BookingStatusPaymentFailed, true},
		{"expired frees slot", models.PaymentStatusExpired, models.BookingStatusPaymentFailed, true},
		{"already cancelled booking", models.PaymentStatusSucceeded, models.BookingStatusCancelled, true},
		{"confirmed keeps slot", models.PaymentStatusSucceeded, models.BookingStatusConfirmed, false},
		{"pending keeps slot", models.PaymentStatusPending, models.BookingStatusPending, false},
		{"no linked booking", models.PaymentStatusFailed, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots := &recordingSlots{}
			bookings := &fakeFinalizerBookings{resultStatus: tt.bookingStatus}
			if err := NewBookingFinalizer(bookings, fakeFinalizerPayments{}, &recordingNotifier{}, slots)(5, tt.paymentStatus); err != nil {
				t.Fatalf("finalize: %v", err)
			}
			if tt.wantInvalidate && (len(slots.providers) != 1 || slots.providers[0] != "prov-1") {
				t.Fatalf("invalidated %v, want [prov-1]", slots.providers)
			}
			if !tt.wantInvalidate && len(slots.providers) != 0 {
				t.Fatalf("unexpected invalidation %v", slots.providers)
			}
		})
	}
}

// A booking lookup failure while invalidating must not fail the webhook: the
// slot cache TTL bounds staleness, and Midtrans must get a 200.
func TestWebhookSucceedsWhenSlotLookupFails(t *testing.T) {
	env := newWebhookEnv(t)
	slots := &recordingSlots{}
	bookings := &fakeFinalizerBookings{resultStatus: models.BookingStatusPaymentFailed, findErr: errors.New("db hiccup")}
	h := &Handlers{repo: env.store, service: NewService(nil, NewMidtransProvider(testServerKey, "", "http://127.0.0.1")),
		finalizeBooking: NewBookingFinalizer(bookings, fakeFinalizerPayments{}, &recordingNotifier{}, slots)}
	engine := newEngine(h)

	w := do(engine, "POST", "/api/payments/webhook", signedBody(t, notificationFor(env.payment.OrderID, "deny", "")), "")
	if w.Code != 200 || decode(t, w)["status"] != "received" {
		t.Fatalf("webhook: %d %s", w.Code, w.Body.String())
	}
	if len(slots.providers) != 0 {
		t.Fatalf("invalidated despite lookup failure: %v", slots.providers)
	}
	if len(env.store.events) != 1 {
		t.Fatalf("webhook event not recorded")
	}
}
