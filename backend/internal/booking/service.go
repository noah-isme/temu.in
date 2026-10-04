package booking

import (
	"context"
	"errors"
	"log/slog"
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
	// ErrNotReschedulable means the booking is missing, not the caller's, or
	// already cancelled or completed at the moment of the write.
	ErrNotReschedulable = errors.New("booking cannot be rescheduled")
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

// bookingStore is the persistence surface the booking service and handlers
// depend on. *Repository satisfies it; tests use an in-memory fake.
type bookingStore interface {
	Create(b *models.Booking) error
	CreateWithPayment(b *models.Booking, p *models.Payment) error
	Reschedule(id, userID uint, date, start, end time.Time) error
	FindByID(id uint) (*models.Booking, error)
	FindByIDForUser(id uint, userID uint) (*models.Booking, error)
	ListByUser(userID uint, limit, offset int) ([]models.Booking, error)
	UpdateStatus(id uint, status string, reason string) error
	CheckConflict(providerID string, date time.Time, startTime, endTime time.Time, excludeID uint) (bool, error)
	GetProviderStats(providerID string) (*models.ProviderStats, error)
	ListByProviderWithNames(providerID, status string, limit, offset int) ([]models.Booking, error)
	ListByUserWithNames(userID uint, limit, offset int) ([]models.Booking, error)
	FindByIDForUserWithNames(id uint, userID uint) (*models.Booking, error)
}

// SlotInvalidator drops cached availability slots for a provider after a
// booking mutation. *availability.Service satisfies it. It returns nothing on
// purpose: invalidation is best-effort and can never turn a successful booking
// mutation into an error (the cache TTL bounds any staleness).
type SlotInvalidator interface {
	InvalidateProviderSlots(ctx context.Context, providerID string)
}

type Service struct {
	repo            bookingStore
	slotInvalidator SlotInvalidator
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

// WithSlotInvalidator sets the optional slot-cache invalidator and returns s.
func (s *Service) WithSlotInvalidator(inv SlotInvalidator) *Service {
	s.slotInvalidator = inv
	return s
}

func (s *Service) invalidateSlots(ctx context.Context, providerID string) {
	if s.slotInvalidator != nil && providerID != "" {
		s.slotInvalidator.InvalidateProviderSlots(ctx, providerID)
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
		slog.WarnContext(ctx, "queue booking confirmation notifications failed", "booking_id", booking.ID, "error", err)
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
		slog.WarnContext(ctx, "queue booking cancellation notifications failed", "booking_id", booking.ID, "error", err)
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
		s.invalidateSlots(ctx, booking.ProviderID)
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
	s.invalidateSlots(ctx, booking.ProviderID)

	s.notifyConfirmation(ctx, booking)

	return booking, nil, nil
}

func (s *Service) GetBookingByID(ctx context.Context, id uint, userID uint) (*models.Booking, error) {
	booking, err := s.repo.FindByIDForUserWithNames(id, userID)
	if err != nil {
		return nil, ErrBookingNotFound
	}
	return booking, nil
}

// clampPage applies the shared list pagination bounds: limit defaults to 20
// and is capped at 100; a negative offset becomes 0.
func clampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (s *Service) ListUserBookings(ctx context.Context, userID uint, limit, offset int) ([]models.Booking, error) {
	limit, offset = clampPage(limit, offset)
	return s.repo.ListByUserWithNames(userID, limit, offset)
}

func isBookingStatus(status string) bool {
	switch status {
	case models.BookingStatusPending, models.BookingStatusConfirmed, models.BookingStatusCompleted,
		models.BookingStatusCancelled, models.BookingStatusPaymentFailed:
		return true
	}
	return false
}

// ListProviderBookings lists a provider's bookings, optionally filtered by
// status (empty = all). An unknown status is ErrInvalidInput. Authorization is
// the caller's responsibility (handlers use auth.RequireProviderAccess).
func (s *Service) ListProviderBookings(ctx context.Context, providerID, status string, limit, offset int) ([]models.Booking, error) {
	if providerID == "" || (status != "" && !isBookingStatus(status)) {
		return nil, ErrInvalidInput
	}
	limit, offset = clampPage(limit, offset)
	return s.repo.ListByProviderWithNames(providerID, status, limit, offset)
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
	s.invalidateSlots(ctx, booking.ProviderID)

	s.notifyCancellation(ctx, booking)

	return nil
}

func (s *Service) RescheduleBooking(ctx context.Context, id uint, userID uint, newDate time.Time, newStartTime, newEndTime time.Time) error {
	booking, err := s.repo.FindByIDForUser(id, userID)
	if err != nil {
		return ErrBookingNotFound
	}

	if booking.Status == models.BookingStatusCancelled || booking.Status == models.BookingStatusCompleted {
		return ErrNotReschedulable
	}

	// The new range must match the booked service's duration, exactly as
	// CreateBooking requires. Bookings without a service (no catalog entry to
	// measure against) keep the free-form range.
	if booking.ServiceID != nil && *booking.ServiceID != "" {
		if s.serviceCatalog == nil {
			return ErrInvalidInput
		}
		selectedService, err := s.serviceCatalog.GetService(*booking.ServiceID)
		if err != nil || selectedService == nil {
			return ErrServiceNotFound
		}
		expectedEnd := newStartTime.Add(time.Duration(selectedService.Duration) * time.Minute)
		if selectedService.Duration <= 0 || !newStartTime.Before(newEndTime) || !expectedEnd.Equal(newEndTime) {
			return ErrInvalidInput
		}
	} else if !newStartTime.Before(newEndTime) {
		return ErrInvalidInput
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

	// One guarded, column-scoped write: it never touches status or payment
	// columns, and fails with ErrNotReschedulable if the booking was cancelled
	// or completed since it was loaded.
	if err := s.repo.Reschedule(id, userID, newDate, newStartTime, newEndTime); err != nil {
		if IsConflictError(err) {
			return ErrBookingConflict
		}
		return err
	}
	s.invalidateSlots(ctx, booking.ProviderID)
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
	s.invalidateSlots(ctx, booking.ProviderID)

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

	if err := s.repo.UpdateStatus(id, models.BookingStatusCompleted, ""); err != nil {
		return err
	}
	s.invalidateSlots(ctx, booking.ProviderID)
	return nil
}

func (s *Service) GetProviderStats(ctx context.Context, providerID string) (*models.ProviderStats, error) {
	return s.repo.GetProviderStats(providerID)
}
