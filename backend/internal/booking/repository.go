package booking

import (
	"errors"
	"gorm.io/gorm"
	"time"

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

// LinkPayment links a booking to a payment by updating the booking's payment_id
func (r *Repository) LinkPayment(bookingID uint, paymentID uint) error {
	return r.db.Model(&models.Booking{}).Where("id = ?", bookingID).Update("payment_id", paymentID).Error
}

// CreateWithPayment creates a payment and booking in a single transaction
// and links the booking to the payment. Returns the created payment.
func (r *Repository) CreateWithPayment(b *models.Booking, amount int64, currency string) (*models.Payment, error) {
	var p models.Payment
	err := r.db.Transaction(func(tx *gorm.DB) error {
		p = models.Payment{
			IntentID:     "pi_" + time.Now().Format("20060102150405"),
			ClientSecret: "cs_" + time.Now().Format("150405"),
			Amount:       amount,
			Currency:     currency,
			Status:       "created",
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		b.Status = "pending"
		if err := tx.Create(b).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Booking{}).Where("id = ?", b.ID).Update("payment_id", p.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetStatusByPayment sets booking status for booking linked to a payment
func (r *Repository) SetStatusByPayment(paymentID uint, status string) error {
	return r.db.Model(&models.Booking{}).Where("payment_id = ?", paymentID).Update("status", status).Error
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
