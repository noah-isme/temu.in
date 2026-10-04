package availability

import (
	"testing"

	"temu.in/backend/internal/models"
)

func TestCalculateSlotsAppliesBlockedException(t *testing.T) {
	schedules := []models.AvailabilitySchedule{{StartTime: "09:00", EndTime: "11:00", IsAvailable: true}}
	exceptions := []models.AvailabilityException{{StartTime: "09:30", EndTime: "10:00", IsAvailable: false}}

	slots := calculateSlots(schedules, exceptions, 30)
	want := []TimeSlot{
		{StartTime: "09:00", EndTime: "09:30"},
		{StartTime: "10:00", EndTime: "10:30"},
		{StartTime: "10:30", EndTime: "11:00"},
	}
	if len(slots) != len(want) {
		t.Fatalf("got %d slots, want %d: %#v", len(slots), len(want), slots)
	}
	for i := range want {
		if slots[i] != want[i] {
			t.Fatalf("slot %d = %#v, want %#v", i, slots[i], want[i])
		}
	}
}

func TestCalculateSlotsFullDayBlock(t *testing.T) {
	schedules := []models.AvailabilitySchedule{{StartTime: "09:00:00", EndTime: "10:00:00", IsAvailable: true}}
	exceptions := []models.AvailabilityException{{IsAvailable: false}}
	if slots := calculateSlots(schedules, exceptions, 30); len(slots) != 0 {
		t.Fatalf("got %#v, want no slots", slots)
	}
}

func TestCalculateSlotsUsesServiceDuration(t *testing.T) {
	schedules := []models.AvailabilitySchedule{{StartTime: "09:00", EndTime: "12:00", IsAvailable: true}}
	slots := calculateSlots(schedules, nil, 60)
	want := []TimeSlot{
		{StartTime: "09:00", EndTime: "10:00"},
		{StartTime: "10:00", EndTime: "11:00"},
		{StartTime: "11:00", EndTime: "12:00"},
	}
	if len(slots) != len(want) {
		t.Fatalf("got %d slots, want %d: %#v", len(slots), len(want), slots)
	}
	for i := range want {
		if slots[i] != want[i] {
			t.Fatalf("slot %d = %#v, want %#v", i, slots[i], want[i])
		}
	}
}

func TestIsValidTimeRejectsInvalidHour(t *testing.T) {
	if isValidTime("29:00") {
		t.Fatal("29:00 must be rejected")
	}
	if !isValidTime("09:30") {
		t.Fatal("09:30 must be accepted")
	}
}

func TestCalculateSlotsEdgeCases(t *testing.T) {
	window := []models.AvailabilitySchedule{{StartTime: "09:00", EndTime: "10:00", IsAvailable: true}}

	t.Run("non-positive duration", func(t *testing.T) {
		if got := calculateSlots(window, nil, 0); got != nil {
			t.Fatalf("got %#v, want nil", got)
		}
		if got := calculateSlots(window, nil, -15); got != nil {
			t.Fatalf("got %#v, want nil", got)
		}
	})
	t.Run("duration longer than window", func(t *testing.T) {
		got := calculateSlots(window, nil, 90)
		if got == nil || len(got) != 0 {
			t.Fatalf("got %#v, want empty non-nil", got)
		}
	})
	t.Run("invalid time strings skipped", func(t *testing.T) {
		schedules := []models.AvailabilitySchedule{
			{StartTime: "nine", EndTime: "10:00", IsAvailable: true},
			{StartTime: "09:00", EndTime: "25:00", IsAvailable: true},
			{StartTime: "11:00", EndTime: "10:00", IsAvailable: true},
		}
		if got := calculateSlots(schedules, nil, 30); len(got) != 0 {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("unavailable schedule ignored", func(t *testing.T) {
		schedules := []models.AvailabilitySchedule{{StartTime: "09:00", EndTime: "10:00", IsAvailable: false}}
		if got := calculateSlots(schedules, nil, 30); len(got) != 0 {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("available exception adds a window", func(t *testing.T) {
		exceptions := []models.AvailabilityException{{StartTime: "18:00:00", EndTime: "19:00:00", IsAvailable: true}}
		got := calculateSlots(nil, exceptions, 30)
		want := []TimeSlot{{StartTime: "18:00", EndTime: "18:30"}, {StartTime: "18:30", EndTime: "19:00"}}
		assertSlots(t, got, want)
	})
	t.Run("available exception without times adds nothing", func(t *testing.T) {
		exceptions := []models.AvailabilityException{{IsAvailable: true}}
		if got := calculateSlots(nil, exceptions, 30); len(got) != 0 {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("overlapping windows de-duplicated and sorted", func(t *testing.T) {
		schedules := []models.AvailabilitySchedule{
			{StartTime: "10:00", EndTime: "11:00", IsAvailable: true},
			{StartTime: "09:00:00", EndTime: "11:00:00", IsAvailable: true},
		}
		got := calculateSlots(schedules, nil, 60)
		want := []TimeSlot{{StartTime: "09:00", EndTime: "10:00"}, {StartTime: "10:00", EndTime: "11:00"}}
		assertSlots(t, got, want)
	})
	t.Run("blocked exception from db seconds keeps adjacent slot", func(t *testing.T) {
		schedules := []models.AvailabilitySchedule{{StartTime: "09:00:00", EndTime: "11:00:00", IsAvailable: true}}
		exceptions := []models.AvailabilityException{{StartTime: "09:00:00", EndTime: "10:00:00", IsAvailable: false}}
		got := calculateSlots(schedules, exceptions, 60)
		assertSlots(t, got, []TimeSlot{{StartTime: "10:00", EndTime: "11:00"}})
	})
	t.Run("full-day block with only start empty", func(t *testing.T) {
		exceptions := []models.AvailabilityException{{EndTime: "10:00", IsAvailable: false}}
		if got := calculateSlots(window, exceptions, 30); len(got) != 0 {
			t.Fatalf("got %#v", got)
		}
	})
}

func TestParseClockLayouts(t *testing.T) {
	for _, in := range []string{"09:00", "09:00:00"} {
		if _, err := parseClock(in); err != nil {
			t.Errorf("parseClock(%q): %v", in, err)
		}
	}
	if _, err := parseClock("9"); err == nil {
		t.Error("parseClock(9) should fail")
	}
}

func assertSlots(t *testing.T, got, want []TimeSlot) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d slots, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slot %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}
