package availability

import (
	"time"

	"temu.in/backend/internal/models"
)

// ScheduleDTO is the API shape of a weekly schedule; times are "HH:MM".
type ScheduleDTO struct {
	ID          string    `json:"id"`
	ProviderID  string    `json:"provider_id"`
	DayOfWeek   int       `json:"day_of_week"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ExceptionDTO is the API shape of a date exception; date is "YYYY-MM-DD" and
// times are "HH:MM", or null for a full-day exception.
type ExceptionDTO struct {
	ID          string    `json:"id"`
	ProviderID  string    `json:"provider_id"`
	Date        string    `json:"date"`
	StartTime   *string   `json:"start_time"`
	EndTime     *string   `json:"end_time"`
	IsAvailable bool      `json:"is_available"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// clockHHMM normalises "HH:MM:SS" (how Postgres `time` columns round-trip) to
// "HH:MM". Values that are not clock strings are returned unchanged.
func clockHHMM(s string) string {
	if parsed, err := parseClock(s); err == nil {
		return parsed.Format("15:04")
	}
	return s
}

func nullableHHMM(s string) *string {
	if s == "" {
		return nil
	}
	v := clockHHMM(s)
	return &v
}

func toScheduleDTO(s models.AvailabilitySchedule) ScheduleDTO {
	return ScheduleDTO{
		ID:          s.ID,
		ProviderID:  s.ProviderID,
		DayOfWeek:   s.DayOfWeek,
		StartTime:   clockHHMM(s.StartTime),
		EndTime:     clockHHMM(s.EndTime),
		IsAvailable: s.IsAvailable,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func toScheduleDTOs(in []models.AvailabilitySchedule) []ScheduleDTO {
	out := make([]ScheduleDTO, 0, len(in))
	for _, s := range in {
		out = append(out, toScheduleDTO(s))
	}
	return out
}

func toExceptionDTO(e models.AvailabilityException) ExceptionDTO {
	return ExceptionDTO{
		ID:          e.ID,
		ProviderID:  e.ProviderID,
		Date:        e.Date.Format("2006-01-02"),
		StartTime:   nullableHHMM(e.StartTime),
		EndTime:     nullableHHMM(e.EndTime),
		IsAvailable: e.IsAvailable,
		Reason:      e.Reason,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toExceptionDTOs(in []models.AvailabilityException) []ExceptionDTO {
	out := make([]ExceptionDTO, 0, len(in))
	for _, e := range in {
		out = append(out, toExceptionDTO(e))
	}
	return out
}
