package booking

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/service"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

// 2030-01-07 is a Monday (day_of_week 1).
const itDate = "2030-01-07"

type itFixture struct {
	db       *gorm.DB
	repo     *Repository
	customer *models.User
	owner    *models.User
	provider *models.ServiceProvider
	service  *models.Service
}

func newITFixture(t *testing.T, price float64) itFixture {
	t.Helper()
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db, "provider")
	customer := testutil.CreateUser(t, db, "user")
	provider := testutil.CreateProvider(t, db, owner.ID)
	svc := testutil.CreateService(t, db, provider.ID, 60, price)
	testutil.CreateSchedule(t, db, provider.ID, 1, "09:00", "17:00")
	return itFixture{db: db, repo: NewRepository(db), customer: customer, owner: owner, provider: provider, service: svc}
}

func (f itFixture) booking(t *testing.T, start, end string) *models.Booking {
	t.Helper()
	return &models.Booking{
		UserID:        f.customer.ID,
		ProviderID:    f.provider.ID,
		ServiceID:     &f.service.ID,
		CustomerName:  "Customer",
		BookingDate:   testutil.MustDate(t, itDate),
		StartTime:     testutil.MustClock(t, start),
		EndTime:       testutil.MustClock(t, end),
		Status:        models.BookingStatusConfirmed,
		PaymentStatus: models.BookingPaymentNotRequired,
	}
}

func (f itFixture) realService() *Service {
	catalog := service.NewService(service.NewRepository(f.db))
	avail := availability.NewService(availability.NewRepository(f.db))
	return NewServiceWithFullDeps(f.repo, nil, avail, nil, nil, nil, catalog, nil)
}

var itSeq int

func newPayment(t *testing.T, userID uint, amount int64) *models.Payment {
	t.Helper()
	itSeq++
	id := fmt.Sprintf("it-%d-%d", time.Now().UnixNano(), itSeq)
	return &models.Payment{
		UserID:       userID,
		Provider:     "midtrans",
		OrderID:      "order-" + id,
		IntentID:     "intent-" + id,
		ClientSecret: "secret-" + id,
		Amount:       amount,
		Currency:     "IDR",
		Status:       models.PaymentStatusPending,
	}
}

func TestIntegrationExclusionConstraint(t *testing.T) {
	t.Run("overlapping bookings conflict", func(t *testing.T) {
		f := newITFixture(t, 0)
		if err := f.repo.Create(f.booking(t, "09:00", "10:00")); err != nil {
			t.Fatalf("first create: %v", err)
		}
		err := f.repo.Create(f.booking(t, "09:30", "10:30"))
		if err == nil || !IsConflictError(err) {
			t.Fatalf("second create err = %v, want exclusion conflict", err)
		}
	})
	t.Run("cancelled booking does not block", func(t *testing.T) {
		f := newITFixture(t, 0)
		first := f.booking(t, "09:00", "10:00")
		first.Status = models.BookingStatusCancelled
		if err := f.repo.Create(first); err != nil {
			t.Fatalf("create cancelled: %v", err)
		}
		if err := f.repo.Create(f.booking(t, "09:00", "10:00")); err != nil {
			t.Fatalf("create over cancelled: %v", err)
		}
	})
	t.Run("adjacent half-open ranges do not conflict", func(t *testing.T) {
		f := newITFixture(t, 0)
		if err := f.repo.Create(f.booking(t, "09:00", "10:00")); err != nil {
			t.Fatal(err)
		}
		if err := f.repo.Create(f.booking(t, "10:00", "11:00")); err != nil {
			t.Fatalf("adjacent create: %v", err)
		}
		if err := f.repo.Create(f.booking(t, "08:00", "09:00")); err != nil {
			t.Fatalf("adjacent before create: %v", err)
		}
	})
	t.Run("other provider same time does not conflict", func(t *testing.T) {
		f := newITFixture(t, 0)
		if err := f.repo.Create(f.booking(t, "09:00", "10:00")); err != nil {
			t.Fatal(err)
		}
		other := testutil.CreateProvider(t, f.db, f.owner.ID)
		b := f.booking(t, "09:00", "10:00")
		b.ProviderID, b.ServiceID = other.ID, nil
		if err := f.repo.Create(b); err != nil {
			t.Fatalf("other provider create: %v", err)
		}
	})
}

func TestIntegrationCheckConflict(t *testing.T) {
	f := newITFixture(t, 0)
	existing := f.booking(t, "10:00", "11:00")
	if err := f.repo.Create(existing); err != nil {
		t.Fatal(err)
	}
	date := testutil.MustDate(t, itDate)
	cases := []struct {
		start, end string
		exclude    uint
		want       bool
	}{
		{"10:30", "11:30", 0, true},
		{"09:00", "10:00", 0, false},
		{"11:00", "12:00", 0, false},
		{"10:00", "11:00", existing.ID, false},
	}
	for _, tc := range cases {
		got, err := f.repo.CheckConflict(f.provider.ID, date, testutil.MustClock(t, tc.start), testutil.MustClock(t, tc.end), tc.exclude)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("CheckConflict(%s-%s, exclude %d) = %v, want %v", tc.start, tc.end, tc.exclude, got, tc.want)
		}
	}
}

func createInput(t *testing.T, f itFixture, start, end string) CreateBookingInput {
	return CreateBookingInput{
		UserID:       f.customer.ID,
		ProviderID:   f.provider.ID,
		ServiceID:    &f.service.ID,
		CustomerName: "Customer",
		BookingDate:  testutil.MustDate(t, itDate),
		StartTime:    testutil.MustClock(t, start),
		EndTime:      testutil.MustClock(t, end),
	}
}

func TestIntegrationServiceCreateBooking(t *testing.T) {
	f := newITFixture(t, 0)
	svc := f.realService()
	ctx := context.Background()

	// 09:00 is the first slot of the schedule window, which Postgres returns
	// as "09:00:00"; it must still be bookable.
	b, p, err := svc.CreateBooking(ctx, createInput(t, f, "09:00", "10:00"))
	if err != nil {
		t.Fatalf("first CreateBooking: %v", err)
	}
	if p != nil || b.Status != models.BookingStatusConfirmed || b.PaymentStatus != models.BookingPaymentNotRequired {
		t.Fatalf("free booking = %+v payment %+v", b, p)
	}

	if _, _, err := svc.CreateBooking(ctx, createInput(t, f, "09:00", "10:00")); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("overlapping CreateBooking err = %v, want ErrBookingConflict", err)
	}
	if _, _, err := svc.CreateBooking(ctx, createInput(t, f, "17:00", "18:00")); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("outside-schedule CreateBooking err = %v, want ErrBookingConflict", err)
	}

	stored, err := svc.GetBookingByID(ctx, b.ID, f.customer.ID)
	if err != nil {
		t.Fatalf("GetBookingByID: %v", err)
	}
	if stored.StartTime.Format("15:04") != "09:00" || stored.EndTime.Format("15:04") != "10:00" || stored.BookingDate.Format("2006-01-02") != itDate {
		t.Fatalf("stored booking times = %s %s-%s", stored.BookingDate, stored.StartTime, stored.EndTime)
	}
	list, err := svc.ListUserBookings(ctx, f.customer.ID, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListUserBookings = %d, %v", len(list), err)
	}
}

func TestIntegrationRescheduleBooking(t *testing.T) {
	f := newITFixture(t, 0)
	svc := f.realService()
	ctx := context.Background()

	target, _, err := svc.CreateBooking(ctx, createInput(t, f, "09:00", "10:00"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CreateBooking(ctx, createInput(t, f, "11:00", "12:00")); err != nil {
		t.Fatal(err)
	}
	date := testutil.MustDate(t, itDate)

	err = svc.RescheduleBooking(ctx, target.ID, f.customer.ID, date, testutil.MustClock(t, "11:00"), testutil.MustClock(t, "12:00"))
	if !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("reschedule onto occupied slot err = %v, want ErrBookingConflict", err)
	}

	if err := svc.RescheduleBooking(ctx, target.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); err != nil {
		t.Fatalf("reschedule onto free slot: %v", err)
	}
	stored, err := f.repo.FindByID(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.StartTime.Format("15:04") != "14:00" || stored.EndTime.Format("15:04") != "15:00" {
		t.Fatalf("rescheduled times = %s-%s", stored.StartTime.Format("15:04"), stored.EndTime.Format("15:04"))
	}

	if err := svc.CancelBooking(ctx, target.ID, f.customer.ID, "changed plans"); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}
	stored, _ = f.repo.FindByID(target.ID)
	if stored.Status != models.BookingStatusCancelled || stored.CancellationReason != "changed plans" {
		t.Fatalf("after cancel = %s / %q", stored.Status, stored.CancellationReason)
	}
}

func TestIntegrationCreateWithPayment(t *testing.T) {
	f := newITFixture(t, 150000)

	b := f.booking(t, "09:00", "10:00")
	p := newPayment(t, f.customer.ID, 150000)
	if err := f.repo.CreateWithPayment(b, p); err != nil {
		t.Fatalf("CreateWithPayment: %v", err)
	}
	stored, err := f.repo.FindByID(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PaymentID == nil || *stored.PaymentID != p.ID || b.PaymentID == nil || *b.PaymentID != p.ID {
		t.Fatalf("payment_id = %v (in-memory %v), want %d", stored.PaymentID, b.PaymentID, p.ID)
	}
	if stored.Status != models.BookingStatusPending || stored.PaymentStatus != models.BookingPaymentPending {
		t.Fatalf("status = %s/%s", stored.Status, stored.PaymentStatus)
	}

	// A conflicting booking rolls back its payment too.
	var before int64
	f.db.Model(&models.Payment{}).Count(&before)
	err = f.repo.CreateWithPayment(f.booking(t, "09:30", "10:30"), newPayment(t, f.customer.ID, 150000))
	if !IsConflictError(err) {
		t.Fatalf("conflicting CreateWithPayment err = %v", err)
	}
	var after int64
	f.db.Model(&models.Payment{}).Count(&after)
	if after != before {
		t.Fatalf("payments %d -> %d: conflicting create must roll back the payment", before, after)
	}

	if err := f.repo.CreateWithPayment(f.booking(t, "11:00", "12:00"), nil); err == nil {
		t.Fatal("nil payment must be rejected")
	}
}

func TestIntegrationSetStatusByPayment(t *testing.T) {
	setup := func(t *testing.T) (itFixture, *models.Booking, *models.Payment) {
		f := newITFixture(t, 150000)
		b := f.booking(t, "09:00", "10:00")
		p := newPayment(t, f.customer.ID, 150000)
		if err := f.repo.CreateWithPayment(b, p); err != nil {
			t.Fatal(err)
		}
		return f, b, p
	}
	reload := func(t *testing.T, f itFixture, id uint) *models.Booking {
		t.Helper()
		b, err := f.repo.FindByID(id)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	t.Run("succeeded confirms and replay is a no-op", func(t *testing.T) {
		f, b, p := setup(t)
		got, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusSucceeded)
		if err != nil || got != models.BookingStatusConfirmed {
			t.Fatalf("SetStatusByPayment = %q, %v", got, err)
		}
		first := reload(t, f, b.ID)
		if first.Status != models.BookingStatusConfirmed || first.PaymentStatus != models.BookingPaymentPaid {
			t.Fatalf("after settle = %s/%s", first.Status, first.PaymentStatus)
		}

		time.Sleep(10 * time.Millisecond)
		got, err = f.repo.SetStatusByPayment(p.ID, models.PaymentStatusSucceeded)
		if err != nil || got != models.BookingStatusConfirmed {
			t.Fatalf("replay SetStatusByPayment = %q, %v", got, err)
		}
		second := reload(t, f, b.ID)
		if !second.UpdatedAt.Equal(first.UpdatedAt) || second.Status != first.Status || second.PaymentStatus != first.PaymentStatus {
			t.Fatalf("replay changed the row: updated_at %s -> %s", first.UpdatedAt, second.UpdatedAt)
		}
	})
	t.Run("failed marks payment_failed", func(t *testing.T) {
		f, b, p := setup(t)
		for _, status := range []string{models.PaymentStatusFailed, models.PaymentStatusExpired, models.PaymentStatusCancelled} {
			got, err := f.repo.SetStatusByPayment(p.ID, status)
			if err != nil || got != models.BookingStatusPaymentFailed {
				t.Fatalf("SetStatusByPayment(%s) = %q, %v", status, got, err)
			}
		}
		stored := reload(t, f, b.ID)
		if stored.Status != models.BookingStatusPaymentFailed || stored.PaymentStatus != models.BookingPaymentFailed {
			t.Fatalf("after failure = %s/%s", stored.Status, stored.PaymentStatus)
		}
	})
	t.Run("pending keeps pending", func(t *testing.T) {
		f, b, p := setup(t)
		got, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusPending)
		if err != nil || got != models.BookingStatusPending {
			t.Fatalf("SetStatusByPayment(pending) = %q, %v", got, err)
		}
		if stored := reload(t, f, b.ID); stored.PaymentStatus != models.BookingPaymentPending {
			t.Fatalf("payment_status = %s", stored.PaymentStatus)
		}
	})
	t.Run("cancelled then settled stays cancelled", func(t *testing.T) {
		f, b, p := setup(t)
		if err := f.repo.UpdateStatus(b.ID, models.BookingStatusCancelled, "customer cancelled"); err != nil {
			t.Fatal(err)
		}
		got, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusSucceeded)
		if err != nil || got != models.BookingStatusCancelled {
			t.Fatalf("SetStatusByPayment = %q, %v; want cancelled", got, err)
		}
		stored := reload(t, f, b.ID)
		if stored.Status != models.BookingStatusCancelled || stored.PaymentStatus != models.BookingPaymentPaid {
			t.Fatalf("after settle = %s/%s, want cancelled/paid", stored.Status, stored.PaymentStatus)
		}
	})
	t.Run("unrecognised status leaves booking untouched", func(t *testing.T) {
		f, b, p := setup(t)
		before := reload(t, f, b.ID)
		got, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusRefunded)
		if err != nil || got != models.BookingStatusPending {
			t.Fatalf("SetStatusByPayment(refunded) = %q, %v", got, err)
		}
		after := reload(t, f, b.ID)
		if !after.UpdatedAt.Equal(before.UpdatedAt) {
			t.Fatal("refunded must not touch the booking row")
		}
	})
	t.Run("unknown payment id", func(t *testing.T) {
		f := newITFixture(t, 0)
		got, err := f.repo.SetStatusByPayment(999999, models.PaymentStatusSucceeded)
		if err != nil || got != "" {
			t.Fatalf("SetStatusByPayment(unknown) = %q, %v", got, err)
		}
	})
}

func TestIntegrationRepositoryReads(t *testing.T) {
	f := newITFixture(t, 150000)
	b := f.booking(t, "09:00", "10:00")
	p := newPayment(t, f.customer.ID, 150000)
	if err := f.repo.CreateWithPayment(b, p); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusSucceeded); err != nil {
		t.Fatal(err)
	}
	pending := f.booking(t, "11:00", "12:00")
	pending.Status = models.BookingStatusPending
	if err := f.repo.Create(pending); err != nil {
		t.Fatal(err)
	}

	byPayment, err := f.repo.FindByPaymentID(p.ID)
	if err != nil {
		t.Fatalf("FindByPaymentID: %v", err)
	}
	if byPayment.User.ID != f.customer.ID || byPayment.ServiceProvider.User.ID != f.owner.ID {
		t.Fatalf("FindByPaymentID preloads = user %d, provider owner %d", byPayment.User.ID, byPayment.ServiceProvider.User.ID)
	}

	if _, err := f.repo.FindByIDForUser(b.ID, f.owner.ID); err == nil {
		t.Fatal("FindByIDForUser must not return another user's booking")
	}

	all, err := f.repo.List(10, 0)
	if err != nil || len(all) != 2 || all[0].ID != pending.ID {
		t.Fatalf("List = %d (first %d), %v", len(all), all[0].ID, err)
	}

	stats, err := f.repo.GetProviderStats(f.provider.ID)
	if err != nil {
		t.Fatalf("GetProviderStats: %v", err)
	}
	if stats.TotalBookings != 2 || stats.PendingBookings != 1 || stats.CompletedBookings != 0 || stats.TotalRevenue != 150000 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestIntegrationListByProviderWithNames(t *testing.T) {
	f := newITFixture(t, 0)
	mk := func(date, start, end, status string) *models.Booking {
		return testutil.CreateBooking(t, f.db, f.customer.ID, f.provider.ID, &f.service.ID, date, start, end, status)
	}
	a := mk("2030-01-07", "09:00", "10:00", models.BookingStatusConfirmed)
	b := mk("2030-01-07", "13:00", "14:00", models.BookingStatusPending)
	c := mk("2030-01-08", "09:00", "10:00", models.BookingStatusCancelled)
	other := testutil.CreateProvider(t, f.db, f.owner.ID)
	testutil.CreateBooking(t, f.db, f.customer.ID, other.ID, nil, "2030-01-07", "09:00", "10:00", models.BookingStatusConfirmed)

	got, err := f.repo.ListByProviderWithNames(f.provider.ID, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []uint{c.ID, b.ID, a.ID}
	if len(got) != len(wantOrder) {
		t.Fatalf("got %d bookings, want %d", len(got), len(wantOrder))
	}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("position %d = booking %d, want %d", i, got[i].ID, id)
		}
		if got[i].Service == nil || got[i].Service.Name != f.service.Name {
			t.Errorf("booking %d service not preloaded: %+v", got[i].ID, got[i].Service)
		}
		if got[i].User.ID != 0 || got[i].User.Email != "" {
			t.Errorf("booking %d must not preload the customer user", got[i].ID)
		}
	}

	pending, err := f.repo.ListByProviderWithNames(f.provider.ID, models.BookingStatusPending, 10, 0)
	if err != nil || len(pending) != 1 || pending[0].ID != b.ID {
		t.Fatalf("status filter = %+v, %v", pending, err)
	}
	paged, err := f.repo.ListByProviderWithNames(f.provider.ID, "", 1, 1)
	if err != nil || len(paged) != 1 || paged[0].ID != b.ID {
		t.Fatalf("limit/offset = %+v, %v", paged, err)
	}

	mine, err := f.repo.ListByUserWithNames(f.customer.ID, 10, 0)
	if err != nil || len(mine) != 4 {
		t.Fatalf("ListByUserWithNames = %d, %v", len(mine), err)
	}
	for _, bk := range mine {
		if bk.ServiceProvider.BusinessName == "" {
			t.Errorf("booking %d provider not preloaded", bk.ID)
		}
	}
	one, err := f.repo.FindByIDForUserWithNames(a.ID, f.customer.ID)
	if err != nil || one.ServiceProvider.BusinessName != f.provider.BusinessName || one.Service == nil {
		t.Fatalf("FindByIDForUserWithNames = %+v, %v", one, err)
	}
	if _, err := f.repo.FindByIDForUserWithNames(a.ID, f.owner.ID); err == nil {
		t.Fatal("FindByIDForUserWithNames must scope by user")
	}
}

type rowStamp struct {
	Count     int64
	UpdatedAt time.Time
}

func stampOf(t *testing.T, db *gorm.DB, table, id string) rowStamp {
	t.Helper()
	var s rowStamp
	if err := db.Table(table).Count(&s.Count).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table(table).Select("updated_at").Where("id = ?", id).Scan(&s.UpdatedAt).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func TestIntegrationRescheduleDoesNotWriteAssociations(t *testing.T) {
	f := newITFixture(t, 0)
	svc := f.realService()
	ctx := context.Background()

	created, _, err := svc.CreateBooking(ctx, createInput(t, f, "09:00", "10:00"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := f.repo.FindByIDForUserWithNames(created.ID, f.customer.ID)
	if err != nil || loaded.Service == nil || loaded.ServiceProvider.ID == "" {
		t.Fatalf("expected populated associations: %+v, %v", loaded, err)
	}

	providerBefore := stampOf(t, f.db, "service_providers", f.provider.ID)
	serviceBefore := stampOf(t, f.db, "services", f.service.ID)
	bookingBefore := stampOf(t, f.db, "bookings", fmt.Sprint(created.ID))
	time.Sleep(10 * time.Millisecond)

	date := testutil.MustDate(t, itDate)
	if err := svc.RescheduleBooking(ctx, created.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); err != nil {
		t.Fatalf("RescheduleBooking: %v", err)
	}

	if after := stampOf(t, f.db, "service_providers", f.provider.ID); after != providerBefore {
		t.Fatalf("service_providers changed: %+v -> %+v", providerBefore, after)
	}
	if after := stampOf(t, f.db, "services", f.service.ID); after != serviceBefore {
		t.Fatalf("services changed: %+v -> %+v", serviceBefore, after)
	}
	bookingAfter := stampOf(t, f.db, "bookings", fmt.Sprint(created.ID))
	if !bookingAfter.UpdatedAt.After(bookingBefore.UpdatedAt) {
		t.Fatalf("bookings.updated_at did not advance: %s -> %s", bookingBefore.UpdatedAt, bookingAfter.UpdatedAt)
	}
}

// Reproduces the lost-update interleaving: the reschedule loads a stale
// snapshot, a payment webhook confirms the booking, then the reschedule
// writes. The webhook's status and payment_status must survive.
func TestIntegrationRescheduleDoesNotRevertConcurrentPayment(t *testing.T) {
	f := newITFixture(t, 150000)
	ctx := context.Background()
	b := f.booking(t, "09:00", "10:00")
	p := newPayment(t, f.customer.ID, 150000)
	if err := f.repo.CreateWithPayment(b, p); err != nil {
		t.Fatal(err)
	}

	// Stale snapshot as the service would load it.
	stale, err := f.repo.FindByIDForUser(b.ID, f.customer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Status != models.BookingStatusPending || stale.PaymentStatus != models.BookingPaymentPending {
		t.Fatalf("stale snapshot = %s/%s", stale.Status, stale.PaymentStatus)
	}

	// Webhook lands between the load and the write.
	if _, err := f.repo.SetStatusByPayment(p.ID, models.PaymentStatusSucceeded); err != nil {
		t.Fatal(err)
	}

	// The service's own load happens before the webhook in the real race; here
	// the repository write is called with the stale view's identity directly.
	date := testutil.MustDate(t, itDate)
	if err := f.repo.Reschedule(stale.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	got, err := f.repo.FindByID(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.BookingStatusConfirmed || got.PaymentStatus != models.BookingPaymentPaid {
		t.Fatalf("webhook state reverted: %s/%s", got.Status, got.PaymentStatus)
	}
	if got.StartTime.Format("15:04") != "14:00" || got.EndTime.Format("15:04") != "15:00" {
		t.Fatalf("times = %s-%s", got.StartTime.Format("15:04"), got.EndTime.Format("15:04"))
	}
	if !got.UpdatedAt.After(stale.UpdatedAt) {
		t.Fatalf("updated_at not bumped: %s -> %s", stale.UpdatedAt, got.UpdatedAt)
	}

	// Same interleaving through the service: the webhook runs inside the
	// availability check, i.e. after the service loaded its snapshot.
	f2 := newITFixture(t, 150000)
	b2 := f2.booking(t, "09:00", "10:00")
	p2 := newPayment(t, f2.customer.ID, 150000)
	if err := f2.repo.CreateWithPayment(b2, p2); err != nil {
		t.Fatal(err)
	}
	svc := f2.realService()
	svc.availabilitySvc = hookedAvailability{inner: svc.availabilitySvc, hook: func() {
		if _, err := f2.repo.SetStatusByPayment(p2.ID, models.PaymentStatusSucceeded); err != nil {
			t.Error(err)
		}
	}}
	if err := svc.RescheduleBooking(ctx, b2.ID, f2.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); err != nil {
		t.Fatalf("RescheduleBooking: %v", err)
	}
	got2, err := f2.repo.FindByID(b2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Status != models.BookingStatusConfirmed || got2.PaymentStatus != models.BookingPaymentPaid || got2.StartTime.Format("15:04") != "14:00" {
		t.Fatalf("after service reschedule = %s/%s %s", got2.Status, got2.PaymentStatus, got2.StartTime.Format("15:04"))
	}
}

type hookedAvailability struct {
	inner AvailabilityService
	hook  func()
}

func (h hookedAvailability) CheckBookingConflicts(providerID string, date time.Time, startTime, endTime string, excludeBookingID ...string) ([]availability.BookingConflict, error) {
	h.hook()
	return h.inner.CheckBookingConflicts(providerID, date, startTime, endTime, excludeBookingID...)
}

func TestIntegrationRescheduleNotReschedulable(t *testing.T) {
	date := testutil.MustDate(t, itDate)
	for _, status := range []string{models.BookingStatusCancelled, models.BookingStatusCompleted} {
		t.Run(status, func(t *testing.T) {
			f := newITFixture(t, 0)
			b := f.booking(t, "09:00", "10:00")
			b.Status = status
			if err := f.repo.Create(b); err != nil {
				t.Fatal(err)
			}
			err := f.realService().RescheduleBooking(context.Background(), b.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00"))
			if !errors.Is(err, ErrNotReschedulable) {
				t.Fatalf("service err = %v, want ErrNotReschedulable", err)
			}
			// The guarded write itself also refuses (status flipped after the load).
			err = f.repo.Reschedule(b.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00"))
			if !errors.Is(err, ErrNotReschedulable) {
				t.Fatalf("repo err = %v, want ErrNotReschedulable", err)
			}
			got, _ := f.repo.FindByID(b.ID)
			if got.Status != status || got.StartTime.Format("15:04") != "09:00" {
				t.Fatalf("booking changed: %s %s", got.Status, got.StartTime.Format("15:04"))
			}
		})
	}
	t.Run("another user's booking", func(t *testing.T) {
		f := newITFixture(t, 0)
		b := f.booking(t, "09:00", "10:00")
		if err := f.repo.Create(b); err != nil {
			t.Fatal(err)
		}
		if err := f.repo.Reschedule(b.ID, f.owner.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); !errors.Is(err, ErrNotReschedulable) {
			t.Fatalf("err = %v, want ErrNotReschedulable", err)
		}
	})
	t.Run("exclusion conflict still surfaces as 23P01", func(t *testing.T) {
		f := newITFixture(t, 0)
		a, other := f.booking(t, "09:00", "10:00"), f.booking(t, "11:00", "12:00")
		if err := f.repo.Create(a); err != nil {
			t.Fatal(err)
		}
		if err := f.repo.Create(other); err != nil {
			t.Fatal(err)
		}
		err := f.repo.Reschedule(a.ID, f.customer.ID, date, testutil.MustClock(t, "11:00"), testutil.MustClock(t, "12:00"))
		if !IsConflictError(err) {
			t.Fatalf("err = %v, want exclusion conflict", err)
		}
	})
}

func TestIntegrationRescheduleEnforcesServiceDuration(t *testing.T) {
	f := newITFixture(t, 0) // service duration is 60 minutes
	svc := f.realService()
	ctx := context.Background()
	b, _, err := svc.CreateBooking(ctx, createInput(t, f, "09:00", "10:00"))
	if err != nil {
		t.Fatal(err)
	}
	date := testutil.MustDate(t, itDate)
	for _, end := range []string{"14:30", "16:00"} {
		err := svc.RescheduleBooking(ctx, b.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, end))
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("14:00-%s err = %v, want ErrInvalidInput", end, err)
		}
	}
	got, _ := f.repo.FindByID(b.ID)
	if got.StartTime.Format("15:04") != "09:00" {
		t.Fatalf("rejected reschedule moved the booking to %s", got.StartTime.Format("15:04"))
	}
	if err := svc.RescheduleBooking(ctx, b.ID, f.customer.ID, date, testutil.MustClock(t, "14:00"), testutil.MustClock(t, "15:00")); err != nil {
		t.Fatalf("exact duration: %v", err)
	}
}

// Postgres accepts "24:00:00" as end_time; reading such a row must work and
// round-trip.
func TestIntegrationEndOfDayBooking(t *testing.T) {
	f := newITFixture(t, 0)
	b := f.booking(t, "23:00", "23:30")
	b.EndTime = models.EndOfDayClock
	if err := f.repo.Create(b); err != nil {
		t.Fatalf("create with 24:00 end: %v", err)
	}
	var raw string
	if err := f.db.Raw("SELECT end_time::text FROM bookings WHERE id = ?", b.ID).Scan(&raw).Error; err != nil || raw != "24:00:00" {
		t.Fatalf("stored end_time = %q, %v", raw, err)
	}
	got, err := f.repo.FindByID(b.ID)
	if err != nil {
		t.Fatalf("read 24:00 row: %v", err)
	}
	if !got.EndTime.Equal(models.EndOfDayClock) || models.FormatClockHM(got.EndTime) != "24:00" {
		t.Fatalf("EndTime = %v", got.EndTime)
	}
	list, err := f.repo.ListByUser(f.customer.ID, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %d, %v", len(list), err)
	}
	if dto := NewCustomerBookingDTO(&list[0]); dto.EndTime != "24:00" {
		t.Fatalf("dto end = %q", dto.EndTime)
	}
	// 24:00 end overlaps a late slot.
	conflict, err := f.repo.CheckConflict(f.provider.ID, testutil.MustDate(t, itDate), testutil.MustClock(t, "23:15"), models.EndOfDayClock, 0)
	if err != nil || !conflict {
		t.Fatalf("CheckConflict(23:15-24:00) = %v, %v", conflict, err)
	}
}

// CheckConflict binds wall-clock strings, so a non-UTC time.Time cannot be
// mis-encoded under the simple protocol.
func TestIntegrationCheckConflictNonUTCClock(t *testing.T) {
	f := newITFixture(t, 0)
	if err := f.repo.Create(f.booking(t, "10:00", "11:00")); err != nil {
		t.Fatal(err)
	}
	jakarta := time.FixedZone("WIB", 7*60*60)
	at := func(h, m int) time.Time { return time.Date(0, 1, 1, h, m, 0, 0, jakarta) }
	date := testutil.MustDate(t, itDate)
	got, err := f.repo.CheckConflict(f.provider.ID, date, at(10, 30), at(11, 30), 0)
	if err != nil || !got {
		t.Fatalf("overlap = %v, %v; want conflict", got, err)
	}
	got, err = f.repo.CheckConflict(f.provider.ID, date, at(11, 0), at(12, 0), 0)
	if err != nil || got {
		t.Fatalf("adjacent = %v, %v; want no conflict", got, err)
	}
}
