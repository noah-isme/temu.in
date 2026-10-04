package booking

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"temu.in/backend/internal/models"
)

type sqlStateError interface {
	SQLState() string
}

func IsConflictError(err error) bool {
	var stateErr sqlStateError
	return errors.As(err, &stateErr) && (stateErr.SQLState() == "23P01" || stateErr.SQLState() == "23505")
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateWithPayment creates a provider-backed payment and booking in a single
// local transaction. The payment has already been prepared by the provider
// service; this method only persists the trusted server-side amount and token.
func (r *Repository) CreateWithPayment(b *models.Booking, p *models.Payment) error {
	if p == nil {
		return errors.New("payment is required")
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		b.Status = models.BookingStatusPending
		b.PaymentStatus = models.BookingPaymentPending
		if err := tx.Create(b).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Booking{}).Where("id = ?", b.ID).Update("payment_id", p.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	b.PaymentID = &p.ID
	return nil
}

// SetStatusByPayment applies a provider payment state to the linked booking
// and keeps booking.status and booking.payment_status in sync. It returns the
// booking's resulting status, or "" when no booking is linked to the payment.
// Only columns whose value actually changes are written, so replaying the same
// webhook is a true no-op (updated_at is not bumped).
func (r *Repository) SetStatusByPayment(paymentID uint, status string) (string, error) {
	var bookingStatus string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var booking models.Booking
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("payment_id = ?", paymentID).First(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Standalone payment intents are valid, but have no booking to update.
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock booking for payment %d: %w", paymentID, err)
		}
		bookingStatus = booking.Status

		var paymentStatus, targetStatus string
		switch status {
		case models.PaymentStatusPending, models.PaymentStatusCreated:
			paymentStatus, targetStatus = models.BookingPaymentPending, models.BookingStatusPending
		case models.PaymentStatusSucceeded, models.BookingStatusConfirmed:
			paymentStatus, targetStatus = models.BookingPaymentPaid, models.BookingStatusConfirmed
		case models.PaymentStatusFailed, models.PaymentStatusExpired, models.PaymentStatusCancelled:
			paymentStatus, targetStatus = models.BookingPaymentFailed, models.BookingStatusPaymentFailed
		default:
			return nil
		}

		updates := map[string]interface{}{}
		if booking.PaymentStatus != paymentStatus {
			updates["payment_status"] = paymentStatus
		}
		if booking.Status != targetStatus && models.CanTransitionBookingStatus(booking.Status, targetStatus) {
			updates["status"] = targetStatus
			bookingStatus = targetStatus
		}
		if len(updates) == 0 {
			return nil
		}
		if err := tx.Model(&models.Booking{}).Where("id = ?", booking.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update booking %d for payment %d: %w", booking.ID, paymentID, err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return bookingStatus, nil
}

func (r *Repository) Create(b *models.Booking) error {
	return r.db.Create(b).Error
}

// Reschedule moves a booking to a new date and time range with one guarded,
// column-scoped UPDATE. Only booking_date, start_time, end_time and updated_at
// are written, so a concurrent status or payment change (webhook, cancel,
// confirm) can never be reverted by a stale snapshot. The statement matches
// only the caller's own booking that is not cancelled or completed; when no
// row matches it returns ErrNotReschedulable. A 23P01 exclusion violation is
// returned unchanged for the caller to map.
func (r *Repository) Reschedule(id, userID uint, date, start, end time.Time) error {
	// Struct + Select (rather than a map) keeps the clock serializer on the
	// time columns and lets GORM bump updated_at.
	res := r.db.Model(&models.Booking{}).
		Where("id = ? AND user_id = ? AND status NOT IN ?", id, userID,
			[]string{models.BookingStatusCancelled, models.BookingStatusCompleted}).
		Select("booking_date", "start_time", "end_time").
		Updates(&models.Booking{BookingDate: date, StartTime: start, EndTime: end})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotReschedulable
	}
	return nil
}

func (r *Repository) List(limit, offset int) ([]models.Booking, error) {
	var out []models.Booking
	if err := r.db.Limit(limit).Offset(offset).Order("id desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListByUser returns bookings for a specific user
func (r *Repository) ListByUser(userID uint, limit, offset int) ([]models.Booking, error) {
	var out []models.Booking
	if err := r.db.Where("user_id = ?", userID).Limit(limit).Offset(offset).Order("id desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// FindByIDForUser finds a booking by id and ensures it belongs to the user
func (r *Repository) FindByIDForUser(id uint, userID uint) (*models.Booking, error) {
	var b models.Booking
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// ListByProviderWithNames returns a provider's bookings (newest date and time
// first) with the Service preloaded for DTO names. status filters when
// non-empty. It is read-only: never pass its results to Update.
func (r *Repository) ListByProviderWithNames(providerID, status string, limit, offset int) ([]models.Booking, error) {
	query := r.db.Preload("Service").Where("provider_id = ?", providerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var out []models.Booking
	if err := query.Order("booking_date desc, start_time desc, id desc").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list bookings for provider: %w", err)
	}
	return out, nil
}

// ListByUserWithNames returns a customer's bookings with the provider and
// service preloaded for DTO names. Read-only: never pass results to Update.
func (r *Repository) ListByUserWithNames(userID uint, limit, offset int) ([]models.Booking, error) {
	var out []models.Booking
	if err := r.db.Preload("ServiceProvider").Preload("Service").Where("user_id = ?", userID).
		Limit(limit).Offset(offset).Order("id desc").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list bookings for user: %w", err)
	}
	return out, nil
}

// FindByIDForUserWithNames loads one of the user's bookings with the provider
// and service preloaded for DTO names. Read-only: never pass it to Update.
func (r *Repository) FindByIDForUserWithNames(id uint, userID uint) (*models.Booking, error) {
	var b models.Booking
	if err := r.db.Preload("ServiceProvider").Preload("Service").Where("id = ? AND user_id = ?", id, userID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) FindByID(id uint) (*models.Booking, error) {
	var b models.Booking
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// FindByPaymentID loads the booking and both notification recipients for a
// payment finalization event. The provider relationship includes its owner so
// provider receipts are always addressed from durable booking data.
func (r *Repository) FindByPaymentID(paymentID uint) (*models.Booking, error) {
	var b models.Booking
	if err := r.db.Preload("User").Preload("ServiceProvider").Preload("ServiceProvider.User").Where("payment_id = ?", paymentID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) UpdateStatus(id uint, status string, reason string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if reason != "" {
		updates["cancellation_reason"] = reason
	}
	return r.db.Model(&models.Booking{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) GetProviderStats(providerID string) (*models.ProviderStats, error) {
	var stats models.ProviderStats

	err := r.db.Model(&models.Booking{}).Where("provider_id = ?", providerID).Count(&stats.TotalBookings).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Booking{}).Where("provider_id = ? AND status = ?", providerID, "pending").Count(&stats.PendingBookings).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Booking{}).Where("provider_id = ? AND status = ?", providerID, "completed").Count(&stats.CompletedBookings).Error
	if err != nil {
		return nil, err
	}

	var totalRevenue int64
	err = r.db.Model(&models.Booking{}).
		Joins("JOIN payments ON bookings.payment_id = payments.id").
		Where("bookings.provider_id = ? AND bookings.status IN (?)", providerID, []string{"confirmed", "completed"}).
		Select("COALESCE(SUM(payments.amount), 0)").
		Scan(&totalRevenue).Error
	if err != nil {
		return nil, err
	}
	stats.TotalRevenue = totalRevenue

	return &stats, nil
}

func (r *Repository) CheckConflict(providerID string, date time.Time, startTime, endTime time.Time, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.Booking{}).
		Where("provider_id = ? AND booking_date = ? AND status NOT IN (?)", providerID, date, []string{"cancelled"}).
		Where("start_time < ? AND end_time > ?", models.FormatClock(endTime), models.FormatClock(startTime))

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
