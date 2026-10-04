package availability

import (
	"os"
	"testing"
	"time"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

func TestIntegrationFullDayExceptionRoundTrip(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	prov := testutil.CreateProvider(t, db, owner.ID)
	repo := NewRepository(db)
	svc := NewService(repo)

	date := testutil.MustDate(t, "2030-03-04")
	exc := &models.AvailabilityException{ProviderID: prov.ID, Date: date, IsAvailable: false, Reason: "holiday"}
	if err := svc.CreateException(exc); err != nil {
		t.Fatalf("create full-day exception: %v", err)
	}

	list, err := svc.GetExceptionsByProviderAndDateRange(prov.ID, date, date)
	if err != nil {
		t.Fatalf("list exceptions: %v", err)
	}
	if len(list) != 1 || list[0].StartTime != "" || list[0].EndTime != "" {
		t.Fatalf("listed exceptions = %+v", list)
	}

	var nulls int64
	if err := db.Raw("SELECT count(*) FROM availability_exceptions WHERE start_time IS NULL AND end_time IS NULL").Scan(&nulls).Error; err != nil {
		t.Fatalf("count nulls: %v", err)
	}
	if nulls != 1 {
		t.Fatalf("full-day exception must store NULL times, got %d rows", nulls)
	}

	// Update a timed exception on a different date back to full-day, so the
	// result is observable independently of the first full-day exception.
	otherDate := testutil.MustDate(t, "2030-03-11") // also a Monday
	timed := &models.AvailabilityException{ProviderID: prov.ID, Date: otherDate, StartTime: "10:00", EndTime: "11:00"}
	if err := svc.CreateException(timed); err != nil {
		t.Fatalf("create timed exception: %v", err)
	}
	if err := svc.UpdateException(timed.ID, prov.ID, &models.AvailabilityException{Date: otherDate, Reason: "all day"}); err != nil {
		t.Fatalf("update to full-day: %v", err)
	}
	var updatedNulls int64
	if err := db.Raw("SELECT count(*) FROM availability_exceptions WHERE id = ? AND start_time IS NULL AND end_time IS NULL", timed.ID).Scan(&updatedNulls).Error; err != nil {
		t.Fatalf("count updated nulls: %v", err)
	}
	if updatedNulls != 1 {
		t.Fatalf("updated exception must have NULL times, got %d matching rows", updatedNulls)
	}

	testutil.CreateSchedule(t, db, prov.ID, int(date.Weekday()), "09:00", "12:00")
	for _, d := range []time.Time{date, otherDate} {
		slots, err := repo.GetAvailableSlots(prov.ID, d, 60)
		if err != nil {
			t.Fatalf("GetAvailableSlots(%s): %v", d.Format("2006-01-02"), err)
		}
		if len(slots) != 0 {
			t.Fatalf("full-day block on %s must remove all slots, got %+v", d.Format("2006-01-02"), slots)
		}
	}
	nextWeek := testutil.MustDate(t, "2030-03-18")
	slots, err := repo.GetAvailableSlots(prov.ID, nextWeek, 60)
	if err != nil || len(slots) != 3 {
		t.Fatalf("unblocked day slots = %+v, %v; want 3", slots, err)
	}
}

func TestIntegrationScheduleIsAvailableFalsePersists(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	prov := testutil.CreateProvider(t, db, owner.ID)
	repo := NewRepository(db)

	for _, available := range []bool{false, true} {
		sched := &models.AvailabilitySchedule{ProviderID: prov.ID, DayOfWeek: 2, StartTime: "09:00", EndTime: "10:00", IsAvailable: available}
		if err := repo.CreateSchedule(sched); err != nil {
			t.Fatalf("CreateSchedule(%v): %v", available, err)
		}
		var stored bool
		if err := db.Raw("SELECT is_available FROM availability_schedules WHERE id = ?", sched.ID).Scan(&stored).Error; err != nil {
			t.Fatal(err)
		}
		if stored != available {
			t.Fatalf("stored is_available = %v, want %v", stored, available)
		}
	}
}

func TestIntegrationAvailableFullDayExceptionRejected(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	prov := testutil.CreateProvider(t, db, owner.ID)
	svc := NewService(NewRepository(db))
	date := testutil.MustDate(t, "2030-03-04")

	if err := svc.CreateException(&models.AvailabilityException{ProviderID: prov.ID, Date: date, IsAvailable: true}); err == nil {
		t.Fatal("available full-day exception must be rejected")
	}
	var n int64
	db.Model(&models.AvailabilityException{}).Count(&n)
	if n != 0 {
		t.Fatalf("rejected exception was stored (%d rows)", n)
	}

	// An available timed window is still accepted, and updating it to an
	// available full-day exception is rejected without changing the row.
	timed := &models.AvailabilityException{ProviderID: prov.ID, Date: date, StartTime: "13:00", EndTime: "14:00", IsAvailable: true}
	if err := svc.CreateException(timed); err != nil {
		t.Fatalf("available timed exception: %v", err)
	}
	if err := svc.UpdateException(timed.ID, prov.ID, &models.AvailabilityException{Date: date, IsAvailable: true}); err == nil {
		t.Fatal("update to available full-day must be rejected")
	}
	var start string
	db.Raw("SELECT start_time::text FROM availability_exceptions WHERE id = ?", timed.ID).Scan(&start)
	if start != "13:00:00" {
		t.Fatalf("row changed: start_time = %q", start)
	}
}

func TestIntegrationScheduleDTONormalisesDBClock(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	prov := testutil.CreateProvider(t, db, owner.ID)
	testutil.CreateSchedule(t, db, prov.ID, 1, "09:00", "17:00")

	schedules, err := NewRepository(db).GetSchedulesByProvider(prov.ID)
	if err != nil || len(schedules) != 1 {
		t.Fatalf("GetSchedulesByProvider = %+v, %v", schedules, err)
	}
	dto := toScheduleDTO(schedules[0])
	if dto.StartTime != "09:00" || dto.EndTime != "17:00" {
		t.Fatalf("DTO times = %q-%q (raw %q-%q)", dto.StartTime, dto.EndTime, schedules[0].StartTime, schedules[0].EndTime)
	}

	date := testutil.MustDate(t, "2030-03-04")
	exc := &models.AvailabilityException{ProviderID: prov.ID, Date: date, StartTime: "12:00", EndTime: "13:00"}
	if err := NewService(NewRepository(db)).CreateException(exc); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, err := NewRepository(db).GetExceptionsByProviderAndDateRange(prov.ID, date, date)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	edto := toExceptionDTO(list[0])
	if edto.Date != "2030-03-04" || edto.StartTime == nil || *edto.StartTime != "12:00" || *edto.EndTime != "13:00" {
		t.Fatalf("exception DTO = %+v", edto)
	}
}
