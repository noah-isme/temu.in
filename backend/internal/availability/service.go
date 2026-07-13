package availability

import (
	"errors"
	"fmt"
	"time"

	"temu.in/backend/internal/models"
)

type Service struct {
	repo RepositoryInterface
}

type RepositoryInterface interface {
	CreateSchedule(schedule *models.AvailabilitySchedule) error
	GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error)
	UpdateSchedule(id string, schedule *models.AvailabilitySchedule) error
	DeleteSchedule(id string) error
	CreateException(exception *models.AvailabilityException) error
	GetExceptionsByProviderAndDateRange(providerID string, startDate, endDate time.Time) ([]models.AvailabilityException, error)
	UpdateException(id string, exception *models.AvailabilityException) error
	DeleteException(id string) error
	GetAvailableSlots(providerID string, date time.Time) ([]TimeSlot, error)
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// Schedule management
func (s *Service) CreateSchedule(schedule *models.AvailabilitySchedule) error {
	// Validate day of week
	if schedule.DayOfWeek < 0 || schedule.DayOfWeek > 6 {
		return errors.New("invalid day of week")
	}

	// Validate time format and logic
	if !isValidTime(schedule.StartTime) || !isValidTime(schedule.EndTime) {
		return errors.New("invalid time format")
	}

	if schedule.StartTime >= schedule.EndTime {
		return errors.New("start time must be before end time")
	}

	return s.repo.CreateSchedule(schedule)
}

func (s *Service) GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error) {
	return s.repo.GetSchedulesByProvider(providerID)
}

func (s *Service) UpdateSchedule(id string, schedule *models.AvailabilitySchedule) error {
	// Similar validation as Create
	if schedule.DayOfWeek < 0 || schedule.DayOfWeek > 6 {
		return errors.New("invalid day of week")
	}

	if !isValidTime(schedule.StartTime) || !isValidTime(schedule.EndTime) {
		return errors.New("invalid time format")
	}

	if schedule.StartTime >= schedule.EndTime {
		return errors.New("start time must be before end time")
	}

	return s.repo.UpdateSchedule(id, schedule)
}

func (s *Service) DeleteSchedule(id string) error {
	return s.repo.DeleteSchedule(id)
}

// Exception management
func (s *Service) CreateException(exception *models.AvailabilityException) error {
	// If times are provided, validate them
	if exception.StartTime != "" && exception.EndTime != "" {
		if !isValidTime(exception.StartTime) || !isValidTime(exception.EndTime) {
			return errors.New("invalid time format")
		}
		if exception.StartTime >= exception.EndTime {
			return errors.New("start time must be before end time")
		}
	}

	return s.repo.CreateException(exception)
}

func (s *Service) GetExceptionsByProviderAndDateRange(providerID string, startDate, endDate time.Time) ([]models.AvailabilityException, error) {
	return s.repo.GetExceptionsByProviderAndDateRange(providerID, startDate, endDate)
}

func (s *Service) UpdateException(id string, exception *models.AvailabilityException) error {
	// Similar validation as Create
	if exception.StartTime != "" && exception.EndTime != "" {
		if !isValidTime(exception.StartTime) || !isValidTime(exception.EndTime) {
			return errors.New("invalid time format")
		}
		if exception.StartTime >= exception.EndTime {
			return errors.New("start time must be before end time")
		}
	}

	return s.repo.UpdateException(id, exception)
}

func (s *Service) DeleteException(id string) error {
	return s.repo.DeleteException(id)
}

// Conflict detection
type BookingConflict struct {
	ProviderID string
	Date       time.Time
	StartTime  string
	EndTime    string
	Reason     string
}

func (s *Service) CheckBookingConflicts(providerID string, date time.Time, startTime, endTime string, excludeBookingID ...string) ([]BookingConflict, error) {
	var conflicts []BookingConflict

	// Get existing bookings for the date (this would need to be implemented in booking service)
	// For now, assume no conflicts from bookings

	// Check availability exceptions
	exceptions, err := s.repo.GetExceptionsByProviderAndDateRange(providerID, date, date)
	if err != nil {
		return nil, err
	}

	for _, exception := range exceptions {
		if !exception.IsAvailable {
			// Check if the booking time overlaps with the unavailable exception
			if timesOverlap(startTime, endTime, exception.StartTime, exception.EndTime) {
				conflicts = append(conflicts, BookingConflict{
					ProviderID: providerID,
					Date:       date,
					StartTime:  exception.StartTime,
					EndTime:    exception.EndTime,
					Reason:     fmt.Sprintf("Exception: %s", exception.Reason),
				})
			}
		}
	}

	// Check against schedule
	dayOfWeek := int(date.Weekday())
	schedules, err := s.repo.GetSchedulesByProvider(providerID)
	if err != nil {
		return nil, err
	}

	available := false
	for _, schedule := range schedules {
		if schedule.DayOfWeek == dayOfWeek && schedule.IsAvailable {
			if timesOverlap(startTime, endTime, schedule.StartTime, schedule.EndTime) {
				available = true
				break
			}
		}
	}

	if !available {
		conflicts = append(conflicts, BookingConflict{
			ProviderID: providerID,
			Date:       date,
			StartTime:  startTime,
			EndTime:    endTime,
			Reason:     "Outside scheduled availability",
		})
	}

	return conflicts, nil
}

// Helper functions
func isValidTime(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func timesOverlap(start1, end1, start2, end2 string) bool {
	// If start2 is empty, it's a full day exception
	if start2 == "" || end2 == "" {
		return true
	}

	return start1 < end2 && start2 < end1
}
