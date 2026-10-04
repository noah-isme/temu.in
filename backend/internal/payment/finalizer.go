package payment

import (
	"context"
	"errors"
	"log/slog"

	"gorm.io/gorm"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/notification"
)

// FinalizerBookings is the booking-repository subset the finalizer needs;
// *booking.Repository satisfies it (booking imports payment, so the
// dependency is expressed as an interface here).
type FinalizerBookings interface {
	SetStatusByPayment(paymentID uint, status string) (string, error)
	FindByPaymentID(paymentID uint) (*models.Booking, error)
}

// FinalizerPayments loads the payment for the receipt; *Repository satisfies it.
type FinalizerPayments interface {
	FindByID(id uint) (*models.Payment, error)
}

// SlotInvalidator drops cached availability slots for a provider;
// *availability.Service satisfies it. Best-effort: it returns nothing.
type SlotInvalidator interface {
	InvalidateProviderSlots(ctx context.Context, providerID string)
}

// NewBookingFinalizer applies a provider payment status to the linked booking
// and, only when the payment succeeded AND the booking actually ended up
// confirmed, enqueues the confirmation and receipt emails. A settled payment
// on an already-cancelled booking therefore notifies nobody. Every effect is
// idempotent (status transitions are no-ops at the same state, notifications
// dedupe on their key), so webhook replays are harmless.
//
// When the booking ends up payment_failed or cancelled its slot is free again,
// so the provider's cached slots are invalidated (slots may be nil). A lookup
// failure there is only logged: the webhook must still succeed, and the cache
// TTL bounds staleness.
func NewBookingFinalizer(bookings FinalizerBookings, payments FinalizerPayments, notifier notification.Service, slots SlotInvalidator) BookingFinalizer {
	return func(paymentID uint, status string) error {
		bookingStatus, err := bookings.SetStatusByPayment(paymentID, status)
		if err != nil {
			return err
		}
		if slots != nil && (bookingStatus == models.BookingStatusPaymentFailed || bookingStatus == models.BookingStatusCancelled) {
			invalidateBookingSlots(bookings, slots, paymentID)
		}
		if status != models.PaymentStatusSucceeded || bookingStatus != models.BookingStatusConfirmed || notifier == nil {
			return nil
		}
		receiptService, ok := notifier.(notification.PaymentReceiptService)
		if !ok {
			return errors.New("payment receipt notification service is unavailable")
		}

		bookingRecord, err := bookings.FindByPaymentID(paymentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		paymentRecord, err := payments.FindByID(paymentID)
		if err != nil {
			return err
		}
		if err := notifier.SendBookingConfirmation(context.Background(), bookingRecord, &bookingRecord.User, &bookingRecord.ServiceProvider); err != nil {
			return err
		}
		return receiptService.SendPaymentReceipt(context.Background(), bookingRecord, &bookingRecord.User, paymentRecord)
	}
}

func invalidateBookingSlots(bookings FinalizerBookings, slots SlotInvalidator, paymentID uint) {
	b, err := bookings.FindByPaymentID(paymentID)
	if err != nil || b == nil {
		slog.Warn("slot invalidation skipped: booking lookup failed", "payment_id", paymentID, "error", err)
		return
	}
	slots.InvalidateProviderSlots(context.Background(), b.ProviderID)
}
