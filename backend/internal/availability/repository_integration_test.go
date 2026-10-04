package availability

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// TestMain lives in integration_test.go. These tests use the per-package
// Postgres database and must not call t.Parallel().

func TestIntegrationScheduleCRUDScopedByProvider(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	mine := testutil.CreateProvider(t, db, testutil.CreateUser(t, db, "provider").ID)
	other := testutil.CreateProvider(t, db, testutil.CreateUser(t, db, "provider").ID)

	s := &models.AvailabilitySchedule{ProviderID: mine.ID, DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", IsAvailable: true}
	if err := repo.CreateSchedule(s); err != nil || s.ID == "" {
		t.Fatalf("create: %v (id %q)", err, s.ID)
	}

	update := &models.AvailabilitySchedule{DayOfWeek: 2, StartTime: "10:00", EndTime: "12:00", IsAvailable: true}
	if err := repo.UpdateSchedule(s.ID, other.ID, update); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update via another provider: err = %v, want ErrRecordNotFound", err)
	}
	if err := repo.DeleteSchedule(s.ID, other.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("delete via another provider: err = %v, want ErrRecordNotFound", err)
	}
	if err := repo.UpdateSchedule(s.ID, mine.ID, update); err != nil {
		t.Fatalf("update: %v", err)
	}
	list, err := repo.GetSchedulesByProvider(mine.ID)
	if err != nil || len(list) != 1 || list[0].DayOfWeek != 2 {
		t.Fatalf("list after update = %+v %v", list, err)
	}
	if dto := toScheduleDTO(list[0]); dto.StartTime != "10:00" || dto.EndTime != "12:00" {
		t.Fatalf("round-tripped times = %q-%q", dto.StartTime, dto.EndTime)
	}
	if others, _ := repo.GetSchedulesByProvider(other.ID); len(others) != 0 {
		t.Fatalf("other provider sees %d schedules", len(others))
	}
	if err := repo.DeleteSchedule(s.ID, mine.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.DeleteSchedule(s.ID, mine.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}
}

func TestIntegrationExceptionsDateRangeInclusive(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	p := testutil.CreateProvider(t, db, testutil.CreateUser(t, db, "provider").ID)
	other := testutil.CreateProvider(t, db, testutil.CreateUser(t, db, "provider").ID)

	for _, d := range []string{"2030-01-01", "2030-01-15", "2030-01-31", "2030-02-01"} {
		if err := repo.CreateException(&models.AvailabilityException{ProviderID: p.ID, Date: testutil.MustDate(t, d), Reason: d}); err != nil {
			t.Fatalf("create %s: %v", d, err)
		}
	}
	got, err := repo.GetExceptionsByProviderAndDateRange(p.ID, testutil.MustDate(t, "2030-01-01"), testutil.MustDate(t, "2030-01-31"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("inclusive range returned %d exceptions, want 3: %+v", len(got), got)
	}

	id := got[0].ID
	if err := repo.UpdateException(id, other.ID, &models.AvailabilityException{Date: testutil.MustDate(t, "2030-01-02")}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update via another provider: err = %v", err)
	}
	if err := repo.DeleteException(id, other.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("delete via another provider: err = %v", err)
	}
	if err := repo.DeleteException(id, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.UpdateException(id, p.ID, &models.AvailabilityException{Date: testutil.MustDate(t, "2030-01-02")}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update deleted: err = %v", err)
	}
}

func TestIntegrationGetAvailableSlotsWithBlockingException(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	p := testutil.CreateProvider(t, db, testutil.CreateUser(t, db, "provider").ID)

	date := testutil.MustDate(t, "2030-01-07") // Monday
	testutil.CreateSchedule(t, db, p.ID, int(date.Weekday()), "09:00", "13:00")
	// Postgres returns these `time` columns as "HH:MM:SS"; the slot
	// computation must parse that form.
	if err := repo.CreateException(&models.AvailabilityException{ProviderID: p.ID, Date: date, StartTime: "10:00", EndTime: "11:00", IsAvailable: false, Reason: "lunch"}); err != nil {
		t.Fatal(err)
	}

	slots, err := repo.GetAvailableSlots(p.ID, date, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"09:00-10:00", "11:00-12:00", "12:00-13:00"}
	if len(slots) != len(want) {
		t.Fatalf("slots = %+v, want %v", slots, want)
	}
	for i, s := range slots {
		if got := s.StartTime + "-" + s.EndTime; got != want[i] {
			t.Fatalf("slot %d = %s, want %s (all %+v)", i, got, want[i], slots)
		}
		if s.Date != "2030-01-07" {
			t.Fatalf("slot date = %q", s.Date)
		}
	}

	// A different weekday has no schedule.
	if none, err := repo.GetAvailableSlots(p.ID, testutil.MustDate(t, "2030-01-08"), 60); err != nil || len(none) != 0 {
		t.Fatalf("tuesday slots = %+v %v", none, err)
	}
}
