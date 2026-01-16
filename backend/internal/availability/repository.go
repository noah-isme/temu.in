package availability

import (
	"time"

	"gorm.io/gorm"
	"temu.in/backend/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateSchedule(schedule *models.AvailabilitySchedule) error {
	return r.db.Create(schedule).Error
}

func (r *Repository) GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error) {
	var schedules []models.AvailabilitySchedule
	err := r.db.Where("provider_id = ?", providerID).Find(&schedules).Error
	return schedules, err
}

func (r *Repository) UpdateSchedule(id string, schedule *models.AvailabilitySchedule) error {
	return r.db.Where("id = ?", id).Updates(schedule).Error
}

func (r *Repository) DeleteSchedule(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.AvailabilitySchedule{}).Error
}

func (r *Repository) CreateException(exception *models.AvailabilityException) error {
	return r.db.Create(exception).Error
}

func (r *Repository) GetExceptionsByProviderAndDateRange(providerID string, startDate, endDate time.Time) ([]models.AvailabilityException, error) {
	var exceptions []models.AvailabilityException
	err := r.db.Where("provider_id = ? AND date >= ? AND date <= ?", providerID, startDate, endDate).Find(&exceptions).Error
	return exceptions, err
}

func (r *Repository) UpdateException(id string, exception *models.AvailabilityException) error {
	return r.db.Where("id = ?", id).Updates(exception).Error
}

func (r *Repository) DeleteException(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.AvailabilityException{}).Error
}

// GetAvailableSlots returns available time slots for a provider on a specific date
func (r *Repository) GetAvailableSlots(providerID string, date time.Time) ([]TimeSlot, error) {
	// Get schedule for the day of week
	dayOfWeek := int(date.Weekday())
	var schedules []models.AvailabilitySchedule
	err := r.db.Where("provider_id = ? AND day_of_week = ? AND is_available = true", providerID, dayOfWeek).Find(&schedules).Error
	if err != nil {
		return nil, err
	}

	// Get exceptions for the date
	var exceptions []models.AvailabilityException
	err = r.db.Where("provider_id = ? AND date = ?", providerID, date).Find(&exceptions).Error
	if err != nil {
		return nil, err
	}

	// TODO: Implement slot calculation logic
	// For now, return empty slots
	return []TimeSlot{}, nil
}

type TimeSlot struct {
	StartTime string
	EndTime   string
}
