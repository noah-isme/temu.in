package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"temu.in/backend/internal/models"
)

// DefaultFrontendBaseURL is used for email links when no base URL is configured.
const DefaultFrontendBaseURL = "http://localhost:5173"

type service struct {
	repo    *Repository
	baseURL string
}

func NewService(repo *Repository) Service {
	return NewServiceWithBaseURL(repo, DefaultFrontendBaseURL)
}

// NewServiceWithBaseURL builds the service with the public frontend origin used
// in password-reset and email-verification links. A trailing slash is dropped
// and an empty value falls back to DefaultFrontendBaseURL.
func NewServiceWithBaseURL(repo *Repository, baseURL string) Service {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultFrontendBaseURL
	}
	return &service{repo: repo, baseURL: baseURL}
}

// frontendLink builds an absolute frontend URL carrying an escaped token.
func (s *service) frontendLink(path, token string) string {
	base := s.baseURL
	if base == "" {
		base = DefaultFrontendBaseURL
	}
	return base + path + "?token=" + url.QueryEscape(token)
}

func (s *service) enqueue(ctx context.Context, notificationType, key, recipientEmail, recipientName, subject string, data map[string]string, bookingID, userID *uint, providerID *string) error {
	if s == nil || s.repo == nil {
		return errors.New("notification service is unavailable")
	}
	if recipientEmail == "" {
		return errors.New("notification recipient email is required")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	return s.repo.Enqueue(ctx, &models.Notification{
		Type:             notificationType,
		Channel:          models.NotificationChannelEmail,
		DeduplicationKey: key,
		RecipientEmail:   recipientEmail,
		RecipientName:    recipientName,
		Subject:          subject,
		Template:         notificationType,
		Payload:          string(payload),
		BookingID:        bookingID,
		UserID:           userID,
		ProviderID:       providerID,
	})
}

func bookingData(booking *models.Booking) map[string]string {
	data := map[string]string{
		"booking_id":    strconv.FormatUint(uint64(booking.ID), 10),
		"customer_name": booking.CustomerName,
		"booking_date":  booking.BookingDate.Format("2006-01-02"),
		"start_time":    models.FormatClockHM(booking.StartTime),
		"end_time":      models.FormatClockHM(booking.EndTime),
	}
	return data
}

func addProviderData(data map[string]string, provider *models.ServiceProvider) {
	if provider != nil {
		data["business_name"] = provider.BusinessName
		data["provider_id"] = provider.ID
	}
}

func providerID(provider *models.ServiceProvider) *string {
	if provider == nil || provider.ID == "" {
		return nil
	}
	id := provider.ID
	return &id
}

func bookingID(booking *models.Booking) *uint {
	if booking == nil || booking.ID == 0 {
		return nil
	}
	id := booking.ID
	return &id
}

func userID(user *models.User) *uint {
	if user == nil || user.ID == 0 {
		return nil
	}
	id := user.ID
	return &id
}

func (s *service) SendBookingConfirmation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	if booking == nil || user == nil || provider == nil {
		return errors.New("booking confirmation data is incomplete")
	}
	data := bookingData(booking)
	addProviderData(data, provider)
	data["recipient_name"] = user.Name
	if err := s.enqueue(ctx, models.NotificationTypeBookingConfirmationCustomer,
		fmt.Sprintf("booking:%d:confirmation:customer", booking.ID), user.Email, user.Name,
		fmt.Sprintf("Booking confirmed with %s", provider.BusinessName), data, bookingID(booking), userID(user), providerID(provider)); err != nil {
		return err
	}

	providerData := bookingData(booking)
	addProviderData(providerData, provider)
	providerData["recipient_name"] = provider.User.Name
	return s.enqueue(ctx, models.NotificationTypeBookingConfirmationProvider,
		fmt.Sprintf("booking:%d:confirmation:provider", booking.ID), provider.User.Email, provider.User.Name,
		"A booking has been confirmed", providerData, bookingID(booking), &provider.UserID, providerID(provider))
}

func (s *service) SendBookingCancellation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	if booking == nil || user == nil || provider == nil {
		return errors.New("booking cancellation data is incomplete")
	}
	data := bookingData(booking)
	addProviderData(data, provider)
	data["reason"] = booking.CancellationReason
	if err := s.enqueue(ctx, models.NotificationTypeBookingCancellationCustomer,
		fmt.Sprintf("booking:%d:cancellation:customer", booking.ID), user.Email, user.Name,
		fmt.Sprintf("Booking cancelled: %s", provider.BusinessName), data, bookingID(booking), userID(user), providerID(provider)); err != nil {
		return err
	}

	providerData := bookingData(booking)
	addProviderData(providerData, provider)
	providerData["reason"] = booking.CancellationReason
	return s.enqueue(ctx, models.NotificationTypeBookingCancellationProvider,
		fmt.Sprintf("booking:%d:cancellation:provider", booking.ID), provider.User.Email, provider.User.Name,
		"A booking has been cancelled", providerData, bookingID(booking), &provider.UserID, providerID(provider))
}

func (s *service) SendBookingReminder(ctx context.Context, booking *models.Booking, user *models.User) error {
	if booking == nil || user == nil {
		return errors.New("booking reminder data is incomplete")
	}
	data := bookingData(booking)
	data["recipient_name"] = user.Name
	return s.enqueue(ctx, models.NotificationTypeBookingReminderCustomer,
		fmt.Sprintf("booking:%d:reminder:customer", booking.ID), user.Email, user.Name,
		"Upcoming booking reminder", data, bookingID(booking), userID(user), nil)
}

func (s *service) SendProviderBookingReminder(ctx context.Context, booking *models.Booking, provider *models.ServiceProvider) error {
	if booking == nil || provider == nil {
		return errors.New("provider booking reminder data is incomplete")
	}
	data := bookingData(booking)
	addProviderData(data, provider)
	return s.enqueue(ctx, models.NotificationTypeBookingReminderProvider,
		fmt.Sprintf("booking:%d:reminder:provider", booking.ID), provider.User.Email, provider.User.Name,
		"Upcoming booking reminder", data, bookingID(booking), &provider.UserID, providerID(provider))
}

func (s *service) SendPaymentReceipt(ctx context.Context, booking *models.Booking, user *models.User, payment *models.Payment) error {
	if booking == nil || user == nil || payment == nil {
		return errors.New("payment receipt data is incomplete")
	}
	data := bookingData(booking)
	data["recipient_name"] = user.Name
	data["amount"] = strconv.FormatInt(payment.Amount, 10)
	data["currency"] = payment.Currency
	data["order_id"] = payment.OrderID
	data["transaction_id"] = payment.TransactionID
	return s.enqueue(ctx, models.NotificationTypePaymentReceipt,
		fmt.Sprintf("booking:%d:payment-receipt", booking.ID), user.Email, user.Name,
		"Payment receipt", data, bookingID(booking), userID(user), nil)
}

// EnqueueUpcomingReminders materializes tomorrow's reminders into the same
// outbox as all other notifications. The deduplication key makes the daily
// scheduler safe to run on every application instance.
func (s *service) EnqueueUpcomingReminders(ctx context.Context, now time.Time) error {
	if s == nil || s.repo == nil || s.repo.db == nil {
		return errors.New("notification service is unavailable")
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	nextDay := dayStart.AddDate(0, 0, 1)
	var bookings []models.Booking
	if err := s.repo.db.WithContext(ctx).
		Preload("User").
		Preload("ServiceProvider").
		Preload("ServiceProvider.User").
		Where("status = ? AND booking_date >= ? AND booking_date < ?", models.BookingStatusConfirmed, dayStart, nextDay).
		Find(&bookings).Error; err != nil {
		return fmt.Errorf("find reminder bookings: %w", err)
	}

	var enqueueErr error
	for i := range bookings {
		booking := &bookings[i]
		if booking.User.Email != "" {
			if err := s.SendBookingReminder(ctx, booking, &booking.User); err != nil {
				enqueueErr = errors.Join(enqueueErr, err)
			}
		}
		if booking.ServiceProvider.User.Email != "" {
			if err := s.SendProviderBookingReminder(ctx, booking, &booking.ServiceProvider); err != nil {
				enqueueErr = errors.Join(enqueueErr, err)
			}
		}
	}
	return enqueueErr
}

func (s *service) SendPasswordReset(ctx context.Context, user *models.User, token string) error {
	if user == nil {
		return errors.New("password reset user is required")
	}
	data := map[string]string{
		"recipient_name": user.Name,
		"reset_link":     s.frontendLink("/reset-password", token),
	}
	return s.enqueue(ctx, models.NotificationTypePasswordReset,
		fmt.Sprintf("user:%d:password-reset:%s", user.ID, token), user.Email, user.Name,
		"Reset your Temu.in password", data, nil, userID(user), nil)
}

func (s *service) SendEmailVerification(ctx context.Context, user *models.User, token string) error {
	if user == nil {
		return errors.New("verification user is required")
	}
	data := map[string]string{
		"recipient_name": user.Name,
		"verify_link":    s.frontendLink("/verify-email", token),
	}
	return s.enqueue(ctx, models.NotificationTypeEmailVerification,
		fmt.Sprintf("user:%d:email-verification:%s", user.ID, token), user.Email, user.Name,
		"Verify your Temu.in email", data, nil, userID(user), nil)
}
