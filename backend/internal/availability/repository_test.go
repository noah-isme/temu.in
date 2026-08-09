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
