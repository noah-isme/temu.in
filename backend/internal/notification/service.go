package notification

import (
	"context"
	"time"

	"temu.in/backend/internal/models"
)

type Service interface {
	SendBookingConfirmation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error
	SendBookingCancellation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error
	SendBookingReminder(ctx context.Context, booking *models.Booking, user *models.User) error
	SendPasswordReset(ctx context.Context, user *models.User, token string) error
	SendEmailVerification(ctx context.Context, user *models.User, token string) error
}

type PaymentReceiptService interface {
	SendPaymentReceipt(ctx context.Context, booking *models.Booking, user *models.User, payment *models.Payment) error
}

type ReminderScheduler interface {
	EnqueueUpcomingReminders(ctx context.Context, now time.Time) error
}
