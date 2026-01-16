package booking

import (
	"context"
	"errors"
	"time"

	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/payment"
)

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrUnauthorized    = errors.New("unauthorized access to booking")
	ErrInvalidInput    = errors.New("invalid booking input")
	ErrSlotUnavailable = errors.New("requested time slot is not available")
	ErrBookingConflict = errors.New("booking conflicts with existing reservation")
)

type CreateBookingInput struct {
	UserID       uint
	ProviderID   string
	ServiceID    *uint
	CustomerName string
	BookingDate  time.Time
	StartTime    time.Time
	EndTime      time.Time
	Notes        string
	Amount       int64
	Currency     string
}

type NotificationService interface {
	SendBookingConfirmation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error
	SendBookingCancellation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error
	SendBookingReminder(ctx context.Context, booking *models.Booking, user *models.User) error
}

type Service struct {
	repo            *Repository
	paymentRepo     *payment.Repository
	availabilitySvc AvailabilityService
	notificationSvc NotificationService
	userRepo        UserRepo
	providerSvc     ProviderService
}

type UserRepo interface {
	FindByID(id uint) (*models.User, error)
}

type ProviderService interface {
	GetProvider(id string) (*models.ServiceProvider, error)
}

func NewService(repo *Repository, paymentRepo *payment.Repository, userRepo UserRepo, providerSvc ProviderService) *Service {
	return &Service{
		repo:        repo,
		paymentRepo: paymentRepo,
		userRepo:    userRepo,
		providerSvc: providerSvc,
	}
}

func NewServiceWithDeps(repo *Repository, paymentRepo *payment.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService) *Service {
	return &Service{
		repo:            repo,
		paymentRepo:     paymentRepo,
		availabilitySvc: availabilitySvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		providerSvc:     providerSvc,
	}
}

func (s *Service) notifyConfirmation(ctx context.Context, booking *models.Booking) {
	if s.notificationSvc == nil {
		return
	}

	user, err := s.userRepo.FindByID(booking.UserID)
	if err != nil {
		return
	}

	provider, err := s.providerSvc.GetProvider(booking.ProviderID)
	if err != nil {
		return
	}

	_ = s.notificationSvc.SendBookingConfirmation(ctx, booking, user, provider)
}

func (s *Service) notifyCancellation(ctx context.Context, booking *models.Booking) {
	if s.notificationSvc == nil {
		return
	}

	user, err := s.userRepo.FindByID(booking.UserID)
	if err != nil {
		return
	}

	provider, err := s.providerSvc.GetProvider(booking.ProviderID)
	if err != nil {
		return
	}

	_ = s.notificationSvc.SendBookingCancellation(ctx, booking, user, provider)
}

type AvailabilityService interface {
	CheckBookingConflicts(providerID string, date time.Time, startTime, endTime string, excludeBookingID ...string) ([]availability.BookingConflict, error)
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (*models.Booking, *models.Payment, error) {
	if input.ProviderID == "" {
		return nil, nil, ErrInvalidInput
	}

	// Check for availability conflicts if availability service is available
	if s.availabilitySvc != nil {
		startTimeStr := input.StartTime.Format("15:04")
		endTimeStr := input.EndTime.Format("15:04")
		conflicts, err := s.availabilitySvc.CheckBookingConflicts(input.ProviderID, input.BookingDate, startTimeStr, endTimeStr)
		if err != nil {
			return nil, nil, err
		}
		if len(conflicts) > 0 {
			return nil, nil, ErrBookingConflict
		}
	}

	// Check for overlapping bookings
	hasConflict, err := s.repo.CheckConflict(input.ProviderID, input.BookingDate, input.StartTime, input.EndTime, 0)
	if err != nil {
		return nil, nil, err
	}
	if hasConflict {
		return nil, nil, ErrBookingConflict
	}

	booking := &models.Booking{
		UserID:        input.UserID,
		ProviderID:    input.ProviderID,
		ServiceID:     input.ServiceID,
		CustomerName:  input.CustomerName,
		BookingDate:   input.BookingDate,
		StartTime:     input.StartTime,
		EndTime:       input.EndTime,
		CustomerNotes: input.Notes,
		Status:        "pending",
		PaymentStatus: "unpaid",
	}

	if input.Amount > 0 {
		currency := input.Currency
		if currency == "" {
			currency = "IDR"
		}
		p, err := s.repo.CreateWithPayment(booking, input.Amount, currency)
		if err != nil {
			return nil, nil, err
		}
		return booking, p, nil
	}

	booking.Status = "confirmed"
	if err := s.repo.Create(booking); err != nil {
		return nil, nil, err
	}

	go s.notifyConfirmation(context.Background(), booking)

	return booking, nil, nil
}

func (s *Service) GetBookingByID(ctx context.Context, id uint, userID uint) (*models.Booking, error) {
	booking, err := s.repo.FindByIDForUser(id, userID)
	if err != nil {
		return nil, ErrBookingNotFound
	}
	return booking, nil
}

func (s *Service) ListUserBookings(ctx context.Context, userID uint, limit, offset int) ([]models.Booking, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListByUser(userID, limit, offset)
}

func (s *Service) CancelBooking(ctx context.Context, id uint, userID uint, reason string) error {
	booking, err := s.repo.FindByIDForUser(id, userID)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status == "cancelled" {
		return nil
	}

	if booking.Status == "completed" {
		return errors.New("cannot cancel completed booking")
	}

	if err := s.repo.UpdateStatus(id, "cancelled", reason); err != nil {
		return err
	}

	go s.notifyCancellation(context.Background(), booking)

	return nil
}

func (s *Service) RescheduleBooking(ctx context.Context, id uint, userID uint, newDate time.Time, newStartTime, newEndTime time.Time) error {
	booking, err := s.repo.FindByIDForUser(id, userID)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status == "cancelled" || booking.Status == "completed" {
		return errors.New("cannot reschedule cancelled or completed booking")
	}

	// Check availability
	if s.availabilitySvc != nil {
		startTimeStr := newStartTime.Format("15:04")
		endTimeStr := newEndTime.Format("15:04")

		conflicts, err := s.availabilitySvc.CheckBookingConflicts(booking.ProviderID, newDate, startTimeStr, endTimeStr)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return ErrBookingConflict
		}
	}

	hasConflict, err := s.repo.CheckConflict(booking.ProviderID, newDate, newStartTime, newEndTime, id)
	if err != nil {
		return err
	}
	if hasConflict {
		return ErrBookingConflict
	}

	booking.BookingDate = newDate
	booking.StartTime = newStartTime
	booking.EndTime = newEndTime

	return s.repo.Update(booking)
}

func (s *Service) ConfirmBooking(ctx context.Context, id uint) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status != "pending" {
		return errors.New("booking is not in pending state")
	}

	if err := s.repo.UpdateStatus(id, "confirmed", ""); err != nil {
		return err
	}

	updatedBooking, err := s.repo.FindByID(id)
	if err == nil {
		go s.notifyConfirmation(context.Background(), updatedBooking)
	}

	return nil
}

func (s *Service) CompleteBooking(ctx context.Context, id uint) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status != "confirmed" {
		return errors.New("booking must be confirmed before completion")
	}

	return s.repo.UpdateStatus(id, "completed", "")
}

func (s *Service) GetProviderStats(ctx context.Context, providerID string) (*models.ProviderStats, error) {
	return s.repo.GetProviderStats(providerID)
}

func (s *Service) LinkPaymentToBooking(ctx context.Context, bookingID uint, clientSecret string, intentID string) error {
	if s.paymentRepo == nil {
		return nil
	}

	var p *models.Payment
	var err error

	if clientSecret != "" {
		p, err = s.paymentRepo.FindByClientSecret(clientSecret)
	} else if intentID != "" {
		p, err = s.paymentRepo.FindByIntent(intentID)
	}

	if err != nil || p == nil {
		return nil
	}

	return s.repo.LinkPayment(bookingID, p.ID)
}
