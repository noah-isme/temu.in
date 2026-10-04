package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"temu.in/backend/internal/models"
)

type Service struct {
	repo  RepositoryInterface
	cache *SlotCache
}

type RepositoryInterface interface {
	CreateSchedule(schedule *models.AvailabilitySchedule) error
	GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error)
	UpdateSchedule(id, providerID string, schedule *models.AvailabilitySchedule) error
	DeleteSchedule(id, providerID string) error
	CreateException(exception *models.AvailabilityException) error
	GetExceptionsByProviderAndDateRange(providerID string, startDate, endDate time.Time) ([]models.AvailabilityException, error)
	UpdateException(id, providerID string, exception *models.AvailabilityException) error
	DeleteException(id, providerID string) error
	GetAvailableSlots(providerID string, date time.Time, durationMinutes int) ([]TimeSlot, error)
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// NewServiceWithCache returns a service that caches computed slots. A nil
// cache (or one over a nil Redis client) disables caching.
func NewServiceWithCache(repo RepositoryInterface, cache *SlotCache) *Service {
	return &Service{repo: repo, cache: cache}
}

// GetAvailableSlots returns the provider's slots for date and duration, served
// from the slot cache when possible. Cache failures fall back to computing.
func (s *Service) GetAvailableSlots(ctx context.Context, providerID string, date time.Time, durationMinutes int) ([]TimeSlot, error) {
	slots, version, hit := s.cache.Get(ctx, providerID, date, durationMinutes)
	if hit {
		return slots, nil
	}
	slots, err := s.repo.GetAvailableSlots(providerID, date, durationMinutes)
	if err != nil {
		return nil, err
	}
	s.cache.Set(ctx, providerID, date, durationMinutes, version, slots)
	return slots, nil
}

// InvalidateProviderSlots drops every cached slot list for the provider. It is
// best-effort: cache errors are logged, never returned.
func (s *Service) InvalidateProviderSlots(ctx context.Context, providerID string) {
	s.cache.Invalidate(ctx, providerID)
}

// invalidate is used by mutators that have no request context; the cache
// bounds the call with its own per-operation timeout.
func (s *Service) invalidate(providerID string) {
	s.cache.Invalidate(context.Background(), providerID)
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

	if err := s.repo.CreateSchedule(schedule); err != nil {
		return err
	}
	s.invalidate(schedule.ProviderID)
	return nil
}

func (s *Service) GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error) {
	return s.repo.GetSchedulesByProvider(providerID)
}

func (s *Service) UpdateSchedule(id, providerID string, schedule *models.AvailabilitySchedule) error {
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

	if err := s.repo.UpdateSchedule(id, providerID, schedule); err != nil {
		return err
	}
	s.invalidate(providerID)
	return nil
}

func (s *Service) DeleteSchedule(id, providerID string) error {
	if err := s.repo.DeleteSchedule(id, providerID); err != nil {
		return err
	}
	s.invalidate(providerID)
	return nil
}

// Exception management
func (s *Service) CreateException(exception *models.AvailabilityException) error {
	if err := validateExceptionTimes(exception); err != nil {
		return err
	}
	// If times are provided, validate them
	if exception.StartTime != "" && exception.EndTime != "" {
		if !isValidTime(exception.StartTime) || !isValidTime(exception.EndTime) {
			return errors.New("invalid time format")
		}
		if exception.StartTime >= exception.EndTime {
			return errors.New("start time must be before end time")
		}
	}

	if err := s.repo.CreateException(exception); err != nil {
		return err
	}
	s.invalidate(exception.ProviderID)
	return nil
}

func (s *Service) GetExceptionsByProviderAndDateRange(providerID string, startDate, endDate time.Time) ([]models.AvailabilityException, error) {
	return s.repo.GetExceptionsByProviderAndDateRange(providerID, startDate, endDate)
}

func (s *Service) UpdateException(id, providerID string, exception *models.AvailabilityException) error {
	if err := validateExceptionTimes(exception); err != nil {
		return err
	}
	// Similar validation as Create
	if exception.StartTime != "" && exception.EndTime != "" {
		if !isValidTime(exception.StartTime) || !isValidTime(exception.EndTime) {
			return errors.New("invalid time format")
		}
		if exception.StartTime >= exception.EndTime {
			return errors.New("start time must be before end time")
		}
	}

	if err := s.repo.UpdateException(id, providerID, exception); err != nil {
		return err
	}
	s.invalidate(providerID)
	return nil
}

func (s *Service) DeleteException(id, providerID string) error {
	if err := s.repo.DeleteException(id, providerID); err != nil {
		return err
	}
	s.invalidate(providerID)
	return nil
}

// validateExceptionTimes requires both times or neither (a full-day
// exception); the table's CHECK constraint enforces the same rule. A full-day
// exception can only block time: an available full-day exception would be a
// silent no-op (slots come from schedules and timed extra windows), so it is
// rejected rather than stored.
func validateExceptionTimes(exception *models.AvailabilityException) error {
	if (exception.StartTime == "") != (exception.EndTime == "") {
		return errors.New("start_time and end_time must both be set or both be empty")
	}
	if exception.IsAvailable && exception.StartTime == "" {
		return errors.New("start_time and end_time are required when is_available is true")
	}
	return nil
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
			if timeRangeWithin(startTime, endTime, schedule.StartTime, schedule.EndTime) {
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

	start1, end1, start2, end2 = clockKey(start1), clockKey(end1), clockKey(start2), clockKey(end2)
	return start1 < end2 && start2 < end1
}

func timeRangeWithin(start, end, windowStart, windowEnd string) bool {
	if windowStart == "" || windowEnd == "" {
		return false
	}
	start, end, windowStart, windowEnd = clockKey(start), clockKey(end), clockKey(windowStart), clockKey(windowEnd)
	return windowStart <= start && end <= windowEnd
}

// clockKey normalises "15:04" and "15:04:05" clock strings to "15:04:05" so
// they compare correctly as strings. Postgres `time` columns round-trip as
// "HH:MM:SS" while request input is "HH:MM"; comparing them raw makes
// "09:00:00" sort after "09:00". Unparseable values are returned unchanged.
func clockKey(value string) string {
	if parsed, err := parseClock(value); err == nil {
		return parsed.Format("15:04:05")
	}
	return value
}
