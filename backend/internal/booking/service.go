package booking

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/payment"
)

var (
	ErrBookingNotFound    = errors.New("booking not found")
	ErrUnauthorized       = errors.New("unauthorized access to booking")
	ErrInvalidInput       = errors.New("invalid booking input")
	ErrServiceNotFound    = errors.New("service not found")
	ErrPaymentUnavailable = errors.New("payment provider is unavailable")
	ErrSlotUnavailable    = errors.New("requested time slot is not available")
	ErrBookingConflict    = errors.New("booking conflicts with existing reservation")
)

type CreateBookingInput struct {
	UserID       uint
	ProviderID   string
	ServiceID    *string
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
	availabilitySvc AvailabilityService
	notificationSvc NotificationService
	userRepo        UserRepo
	providerSvc     ProviderService
	serviceCatalog  ServiceCatalog
	paymentCreator  PaymentCreator
}

type UserRepo interface {
	FindByID(id uint) (*models.User, error)
}

type ProviderService interface {
	GetProvider(id string) (*models.ServiceProvider, error)
}

type ServiceCatalog interface {
	GetService(id string) (*models.Service, error)
}

type PaymentCreator interface {
	Prepare(ctx context.Context, req payment.CreateRequest) (*models.Payment, error)
}

func NewService(repo *Repository, paymentRepo *payment.Repository, userRepo UserRepo, providerSvc ProviderService) *Service {
	return NewServiceWithFullDeps(repo, paymentRepo, nil, nil, userRepo, providerSvc, nil, nil)
}

func NewServiceWithDeps(repo *Repository, paymentRepo *payment.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService) *Service {
	return NewServiceWithFullDeps(repo, paymentRepo, availabilitySvc, notificationSvc, userRepo, providerSvc, nil, nil)
}

func NewServiceWithFullDeps(repo *Repository, paymentRepo *payment.Repository, availabilitySvc AvailabilityService, notificationSvc NotificationService, userRepo UserRepo, providerSvc ProviderService, serviceCatalog ServiceCatalog, paymentCreator PaymentCreator) *Service {
	return &Service{
		repo:            repo,
		availabilitySvc: availabilitySvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		providerSvc:     providerSvc,
		serviceCatalog:  serviceCatalog,
		paymentCreator:  paymentCreator,
	}
}

func (s *Service) notifyConfirmation(ctx context.Context, booking *models.Booking) {
	if s.notificationSvc == nil || s.userRepo == nil || s.providerSvc == nil {
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

	if err := s.notificationSvc.SendBookingConfirmation(ctx, booking, user, provider); err != nil {
		log.Printf("queue booking confirmation notifications: %v", err)
	}
}

func (s *Service) notifyCancellation(ctx context.Context, booking *models.Booking) {
	if s.notificationSvc == nil || s.userRepo == nil || s.providerSvc == nil {
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

	if err := s.notificationSvc.SendBookingCancellation(ctx, booking, user, provider); err != nil {
		log.Printf("queue booking cancellation notifications: %v", err)
	}
}

type AvailabilityService interface {
	CheckBookingConflicts(providerID string, date time.Time, startTime, endTime string, excludeBookingID ...string) ([]availability.BookingConflict, error)
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (*models.Booking, *models.Payment, error) {
	if input.ProviderID == "" || input.ServiceID == nil || *input.ServiceID == "" || s.serviceCatalog == nil {
		return nil, nil, ErrInvalidInput
	}

	selectedService, err := s.serviceCatalog.GetService(*input.ServiceID)
	if err != nil || selectedService == nil {
		return nil, nil, ErrServiceNotFound
	}
	if !selectedService.IsActive || selectedService.ProviderID != input.ProviderID || selectedService.Duration <= 0 || selectedService.Price < 0 || math.IsNaN(selectedService.Price) || math.IsInf(selectedService.Price, 0) {
		return nil, nil, ErrInvalidInput
	}
	expectedEnd := input.StartTime.Add(time.Duration(selectedService.Duration) * time.Minute)
	if !input.StartTime.Before(input.EndTime) || !expectedEnd.Equal(input.EndTime) {
		return nil, nil, ErrInvalidInput
	}
	amountFloat := math.Round(selectedService.Price)
	if amountFloat > float64(math.MaxInt64) || (selectedService.Price > 0 && amountFloat <= 0) {
		return nil, nil, ErrInvalidInput
	}
	amount := int64(amountFloat)

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
		Status:        models.BookingStatusPending,
		PaymentStatus: models.BookingPaymentUnpaid,
	}

	if amount > 0 {
		if s.paymentCreator == nil {
			return nil, nil, ErrPaymentUnavailable
		}
		p, err := s.paymentCreator.Prepare(ctx, payment.CreateRequest{
			UserID:      input.UserID,
			ProviderID:  input.ProviderID,
			ServiceID:   *input.ServiceID,
			Amount:      amount,
			Currency:    "IDR",
			Description: selectedService.Name,
		})
		if err != nil {
			return nil, nil, err
		}
		if p == nil || p.Amount != amount || p.Currency != "IDR" {
			return nil, nil, ErrPaymentUnavailable
		}
		p.UserID = input.UserID
		if err := s.repo.CreateWithPayment(booking, p); err != nil {
			if IsConflictError(err) {
				return nil, nil, ErrBookingConflict
			}
			return nil, nil, err
		}
		return booking, p, nil
	}

	booking.Status = models.BookingStatusConfirmed
	booking.PaymentStatus = models.BookingPaymentNotRequired
	if err := s.repo.Create(booking); err != nil {
		if IsConflictError(err) {
			return nil, nil, ErrBookingConflict
		}
		return nil, nil, err
	}

	s.notifyConfirmation(ctx, booking)

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

	if booking.Status == models.BookingStatusCancelled {
		return nil
	}

	if booking.Status == models.BookingStatusCompleted {
		return errors.New("cannot cancel completed booking")
	}

	if err := s.repo.UpdateStatus(id, models.BookingStatusCancelled, reason); err != nil {
		return err
	}

	s.notifyCancellation(ctx, booking)

	return nil
}

func (s *Service) RescheduleBooking(ctx context.Context, id uint, userID uint, newDate time.Time, newStartTime, newEndTime time.Time) error {
	booking, err := s.repo.FindByIDForUser(id, userID)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status == models.BookingStatusCancelled || booking.Status == models.BookingStatusCompleted {
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

	if err := s.repo.Update(booking); err != nil {
		if IsConflictError(err) {
			return ErrBookingConflict
		}
		return err
	}
	return nil
}

func (s *Service) ConfirmBooking(ctx context.Context, id uint) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status != models.BookingStatusPending {
		return errors.New("booking is not in pending state")
	}
	if booking.PaymentID != nil && booking.PaymentStatus != models.BookingPaymentPaid {
		return errors.New("payment must succeed before confirming booking")
	}

	if err := s.repo.UpdateStatus(id, models.BookingStatusConfirmed, ""); err != nil {
		return err
	}

	updatedBooking, err := s.repo.FindByID(id)
	if err == nil {
		s.notifyConfirmation(ctx, updatedBooking)
	}

	return nil
}

func (s *Service) CompleteBooking(ctx context.Context, id uint) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status != models.BookingStatusConfirmed {
		return errors.New("booking must be confirmed before completion")
	}
	if booking.PaymentID != nil && booking.PaymentStatus != models.BookingPaymentPaid {
		return errors.New("payment must succeed before completion")
	}

	return s.repo.UpdateStatus(id, models.BookingStatusCompleted, "")
}

func (s *Service) GetProviderStats(ctx context.Context, providerID string) (*models.ProviderStats, error) {
	return s.repo.GetProviderStats(providerID)
}
