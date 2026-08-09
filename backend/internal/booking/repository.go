package booking

import (
	"errors"
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

// SetStatusByPayment applies a provider state to the linked booking and keeps
// booking.status and booking.payment_status in sync. Replaying a webhook is a
// no-op once the booking has reached the same state.
func (r *Repository) SetStatusByPayment(paymentID uint, status string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var booking models.Booking
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("payment_id = ?", paymentID).First(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Standalone payment intents are valid, but have no booking to update.
			return nil
		}
		if err != nil {
			return err
		}

		updates := map[string]interface{}{}
		switch status {
		case models.PaymentStatusPending, models.PaymentStatusCreated:
			updates["payment_status"] = models.BookingPaymentPending
			if models.CanTransitionBookingStatus(booking.Status, models.BookingStatusPending) {
				updates["status"] = models.BookingStatusPending
			}
		case models.PaymentStatusSucceeded, models.BookingStatusConfirmed:
			updates["payment_status"] = models.BookingPaymentPaid
			if models.CanTransitionBookingStatus(booking.Status, models.BookingStatusConfirmed) || booking.Status == models.BookingStatusConfirmed {
				updates["status"] = models.BookingStatusConfirmed
			}
		case models.PaymentStatusFailed, models.PaymentStatusExpired, models.PaymentStatusCancelled:
			updates["payment_status"] = models.BookingPaymentFailed
			if models.CanTransitionBookingStatus(booking.Status, models.BookingStatusPaymentFailed) || booking.Status == models.BookingStatusPaymentFailed {
				updates["status"] = models.BookingStatusPaymentFailed
			}
		default:
			return nil
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&models.Booking{}).Where("id = ?", booking.ID).Updates(updates).Error
	})
}

func (r *Repository) Create(b *models.Booking) error {
	return r.db.Create(b).Error
}

func (r *Repository) Update(b *models.Booking) error {
	return r.db.Save(b).Error
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
		Where("start_time < ? AND end_time > ?", endTime, startTime)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
