package notification

import (
	"context"
	"fmt"
	"log"

	"temu.in/backend/internal/models"
)

type service struct {
	emailProvider EmailProvider
}

func NewService(emailProvider EmailProvider) Service {
	return &service{
		emailProvider: emailProvider,
	}
}

func (s *service) SendBookingConfirmation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	// 1. Notify Customer
	customerSubject := fmt.Sprintf("Booking Confirmed with %s", provider.BusinessName)
	customerBody := fmt.Sprintf("Hi %s,\n\nYour booking with %s is confirmed for %s at %s.\n\nThank you for choosing Temu.in!",
		user.Name, provider.BusinessName, booking.BookingDate.Format("2006-01-02"), booking.StartTime.Format("15:04"))

	if err := s.emailProvider.SendEmail([]string{user.Email}, customerSubject, customerBody); err != nil {
		log.Printf("Failed to send customer confirmation email: %v", err)
		// Don't fail the request, just log
	}

	// 2. Notify Provider (if provider has email, assumed via some mechanism, currently provider linked to user)
	// We need provider's user email. We assume provider struct has it or we fetch it.
	// For now, let's assume provider struct doesn't have email directly, but we passed provider model.
	// Provider model has UserID. We'd need to fetch user.
	// To keep it simple, we skip provider notification or assume provider email is passed.
	// Let's update interface to accept providerUser? Or just skip for now.

	return nil
}

func (s *service) SendBookingCancellation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	subject := fmt.Sprintf("Booking Cancelled: %s", provider.BusinessName)
	body := fmt.Sprintf("Hi %s,\n\nYour booking with %s for %s has been cancelled.\nReason: %s\n\nTemu.in",
		user.Name, provider.BusinessName, booking.BookingDate.Format("2006-01-02"), booking.CancellationReason)

	if err := s.emailProvider.SendEmail([]string{user.Email}, subject, body); err != nil {
		log.Printf("Failed to send cancellation email: %v", err)
	}
	return nil
}

func (s *service) SendBookingReminder(ctx context.Context, booking *models.Booking, user *models.User) error {
	// Used for async jobs
	subject := "Upcoming Booking Reminder"
	body := fmt.Sprintf("Hi %s,\n\nReminder: You have a booking tomorrow at %s.\n\nTemu.in",
		user.Name, booking.StartTime.Format("15:04"))

	return s.emailProvider.SendEmail([]string{user.Email}, subject, body)
}

func (s *service) SendPasswordReset(ctx context.Context, user *models.User, token string) error {
	subject := "Reset Your Password - Temu.in"
	// In production, this would be a link to the frontend
	resetLink := fmt.Sprintf("http://localhost:5173/reset-password?token=%s", token)
	body := fmt.Sprintf("Hi %s,\n\nYou requested a password reset. Click the link below to reset your password:\n\n%s\n\nIf you did not request this, please ignore this email.\n\nTemu.in",
		user.Name, resetLink)

	return s.emailProvider.SendEmail([]string{user.Email}, subject, body)
}

func (s *service) SendEmailVerification(ctx context.Context, user *models.User, token string) error {
	subject := "Verify Your Email - Temu.in"
	// In production, this would be a link to the frontend
	verifyLink := fmt.Sprintf("http://localhost:5173/verify-email?token=%s", token)
	body := fmt.Sprintf("Hi %s,\n\nWelcome to Temu.in! Please verify your email address by clicking the link below:\n\n%s\n\nTemu.in",
		user.Name, verifyLink)

	return s.emailProvider.SendEmail([]string{user.Email}, subject, body)
}
