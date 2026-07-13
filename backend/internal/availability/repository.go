package availability

import (
	"sort"
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
	err = r.db.Where("provider_id = ? AND date = ?", providerID, date.Format("2006-01-02")).Find(&exceptions).Error
	if err != nil {
		return nil, err
	}

	return calculateSlots(schedules, exceptions, 30), nil
}

type TimeSlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func calculateSlots(schedules []models.AvailabilitySchedule, exceptions []models.AvailabilityException, durationMinutes int) []TimeSlot {
	if durationMinutes <= 0 {
		durationMinutes = 30
	}

	windows := make([]TimeSlot, 0, len(schedules)+len(exceptions))
	for _, schedule := range schedules {
		if schedule.IsAvailable {
			windows = append(windows, TimeSlot{StartTime: schedule.StartTime, EndTime: schedule.EndTime})
		}
	}
	for _, exception := range exceptions {
		if exception.IsAvailable && exception.StartTime != "" && exception.EndTime != "" {
			windows = append(windows, TimeSlot{StartTime: exception.StartTime, EndTime: exception.EndTime})
		}
	}

	result := make([]TimeSlot, 0)
	seen := make(map[string]struct{})
	step := time.Duration(durationMinutes) * time.Minute
	for _, window := range windows {
		start, startErr := parseClock(window.StartTime)
		end, endErr := parseClock(window.EndTime)
		if startErr != nil || endErr != nil || !start.Before(end) {
			continue
		}
		for cursor := start; !cursor.Add(step).After(end); cursor = cursor.Add(step) {
			slotEnd := cursor.Add(step)
			startText := cursor.Format("15:04")
			endText := slotEnd.Format("15:04")
			blocked := false
			for _, exception := range exceptions {
				if exception.IsAvailable {
					continue
				}
				if exception.StartTime == "" || exception.EndTime == "" || timesOverlap(startText, endText, exception.StartTime, exception.EndTime) {
					blocked = true
					break
				}
			}
			if !blocked {
				key := startText + "-" + endText
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					result = append(result, TimeSlot{StartTime: startText, EndTime: endText})
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].StartTime < result[j].StartTime })
	return result
}

func parseClock(value string) (time.Time, error) {
	for _, layout := range []string{"15:04", "15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, &time.ParseError{Value: value, Layout: "15:04"}
}
