package booking

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/models"
)

const testUserID = uint(7)

type serviceDeps struct {
	store    *fakeStore
	avail    *fakeAvailability
	catalog  *fakeCatalog
	provider *fakeProviderSvc
	users    *fakeUserRepo
	notifier *fakeNotifier
	payments *fakePaymentCreator
}

func newTestService(price float64, seed ...models.Booking) (*Service, *serviceDeps) {
	d := &serviceDeps{
		store: newFakeStore(seed...),
		avail: &fakeAvailability{},
		catalog: &fakeCatalog{services: map[string]*models.Service{
			testServiceID: {ID: testServiceID, ProviderID: testProviderID, Name: "Cut", Duration: 60, Price: price, IsActive: true},
		}},
		provider: &fakeProviderSvc{providers: map[string]*models.ServiceProvider{testProviderID: {ID: testProviderID, BusinessName: "Shop"}}},
		users:    &fakeUserRepo{users: map[uint]*models.User{testUserID: {ID: testUserID, Name: "Ann"}}},
		notifier: &fakeNotifier{},
		payments: &fakePaymentCreator{},
	}
	svc := &Service{
		repo:            d.store,
		availabilitySvc: d.avail,
		notificationSvc: d.notifier,
		userRepo:        d.users,
		providerSvc:     d.provider,
		serviceCatalog:  d.catalog,
		paymentCreator:  d.payments,
	}
	return svc, d
}

func validInput() CreateBookingInput {
	sid := testServiceID
	date := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	start := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	return CreateBookingInput{
		UserID:       testUserID,
		ProviderID:   testProviderID,
		ServiceID:    &sid,
		CustomerName: "Ann",
		BookingDate:  date,
		StartTime:    start,
		EndTime:      start.Add(time.Hour),
	}
}

func TestCreateBookingInvalidInput(t *testing.T) {
	ctx := context.Background()
	empty := ""

	cases := map[string]func(*Service, *serviceDeps, *CreateBookingInput){
		"empty provider": func(_ *Service, _ *serviceDeps, in *CreateBookingInput) { in.ProviderID = "" },
		"nil service":    func(_ *Service, _ *serviceDeps, in *CreateBookingInput) { in.ServiceID = nil },
		"empty service":  func(_ *Service, _ *serviceDeps, in *CreateBookingInput) { in.ServiceID = &empty },
		"nil catalog":    func(s *Service, _ *serviceDeps, _ *CreateBookingInput) { s.serviceCatalog = nil },
		"inactive service": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) {
			d.catalog.services[testServiceID].IsActive = false
		},
		"provider mismatch": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) {
			d.catalog.services[testServiceID].ProviderID = "other"
		},
		"zero duration": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) {
			d.catalog.services[testServiceID].Duration = 0
		},
		"negative price": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) { d.catalog.services[testServiceID].Price = -1 },
		"NaN price": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) {
			d.catalog.services[testServiceID].Price = math.NaN()
		},
		"Inf price": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) {
			d.catalog.services[testServiceID].Price = math.Inf(1)
		},
		"tiny price rounds 0": func(_ *Service, d *serviceDeps, _ *CreateBookingInput) { d.catalog.services[testServiceID].Price = 0.2 },
		"end != start+dur": func(_ *Service, _ *serviceDeps, in *CreateBookingInput) {
			in.EndTime = in.StartTime.Add(30 * time.Minute)
		},
		"end before start": func(_ *Service, _ *serviceDeps, in *CreateBookingInput) { in.EndTime = in.StartTime.Add(-time.Hour) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			svc, d := newTestService(0)
			in := validInput()
			mutate(svc, d, &in)
			b, p, err := svc.CreateBooking(ctx, in)
			if !errors.Is(err, ErrInvalidInput) || b != nil || p != nil {
				t.Fatalf("got %v, %v, %v; want ErrInvalidInput", b, p, err)
			}
			if d.store.called("Create")+d.store.called("CreateWithPayment") != 0 {
				t.Fatal("store must not be written")
			}
		})
	}
}

func TestCreateBookingCatalogErrorIsServiceNotFound(t *testing.T) {
	svc, d := newTestService(0)
	d.catalog.err = errFake
	if _, _, err := svc.CreateBooking(context.Background(), validInput()); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("err = %v", err)
	}
	svc, _ = newTestService(0)
	in := validInput()
	other := "missing"
	in.ServiceID = &other
	if _, _, err := svc.CreateBooking(context.Background(), in); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("unknown service err = %v", err)
	}
}

func TestCreateBookingConflicts(t *testing.T) {
	ctx := context.Background()

	svc, d := newTestService(0)
	d.avail.conflicts = []availability.BookingConflict{{}}
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("availability conflict err = %v", err)
	}

	svc, d = newTestService(0)
	d.avail.err = errFake
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, errFake) {
		t.Fatalf("availability error err = %v", err)
	}

	svc, d = newTestService(0)
	d.store.conflict = true
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("CheckConflict=true err = %v", err)
	}
	if d.store.called("Create") != 0 {
		t.Fatal("Create must not be called on conflict")
	}

	svc, d = newTestService(0)
	d.store.checkConflictErr = errFake
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, errFake) {
		t.Fatalf("CheckConflict error err = %v", err)
	}

	svc, d = newTestService(0)
	d.store.createErr = sqlStateErr{"23P01"}
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("23P01 on Create err = %v", err)
	}

	svc, d = newTestService(0)
	d.store.createErr = errFake
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, errFake) {
		t.Fatalf("Create error err = %v", err)
	}

	svc, d = newTestService(50000)
	d.store.createWithPaymentErr = sqlStateErr{"23P01"}
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("23P01 on CreateWithPayment err = %v", err)
	}

	svc, d = newTestService(50000)
	d.store.createWithPaymentErr = errFake
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, errFake) {
		t.Fatalf("CreateWithPayment error err = %v", err)
	}
}

func TestCreateBookingFreeService(t *testing.T) {
	svc, d := newTestService(0)
	b, p, err := svc.CreateBooking(context.Background(), validInput())
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if p != nil {
		t.Fatalf("free booking returned payment %+v", p)
	}
	if d.store.called("Create") != 1 || d.store.called("CreateWithPayment") != 0 {
		t.Fatalf("calls = %v", d.store.calls)
	}
	if b.Status != models.BookingStatusConfirmed || b.PaymentStatus != models.BookingPaymentNotRequired {
		t.Fatalf("status = %s/%s", b.Status, b.PaymentStatus)
	}
	if d.notifier.confirmations != 1 {
		t.Fatalf("confirmations = %d, want 1", d.notifier.confirmations)
	}
	if d.payments.calls != 0 {
		t.Fatal("payment creator must not be called for free service")
	}
}

func TestCreateBookingFreeServiceNotificationFailuresDoNotFail(t *testing.T) {
	svc, d := newTestService(0)
	d.notifier.err = errFake
	if _, _, err := svc.CreateBooking(context.Background(), validInput()); err != nil {
		t.Fatalf("notifier error must not fail booking: %v", err)
	}
	svc, d = newTestService(0)
	d.users.err = errFake
	if _, _, err := svc.CreateBooking(context.Background(), validInput()); err != nil || d.notifier.confirmations != 0 {
		t.Fatalf("user lookup failure: err=%v confirmations=%d", err, d.notifier.confirmations)
	}
	svc, d = newTestService(0)
	d.provider.err = errFake
	if _, _, err := svc.CreateBooking(context.Background(), validInput()); err != nil || d.notifier.confirmations != 0 {
		t.Fatalf("provider lookup failure: err=%v confirmations=%d", err, d.notifier.confirmations)
	}
	svc, _ = newTestService(0)
	svc.notificationSvc = nil
	if _, _, err := svc.CreateBooking(context.Background(), validInput()); err != nil {
		t.Fatalf("nil notifier: %v", err)
	}
}

func TestCreateBookingPaidService(t *testing.T) {
	ctx := context.Background()

	svc, _ := newTestService(50000)
	svc.paymentCreator = nil
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrPaymentUnavailable) {
		t.Fatalf("nil payment creator err = %v", err)
	}

	svc, d := newTestService(50000)
	d.payments.payment = &models.Payment{Amount: 1, Currency: "IDR"}
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrPaymentUnavailable) {
		t.Fatalf("mismatched amount err = %v", err)
	}

	svc, d = newTestService(50000)
	d.payments.payment = &models.Payment{Amount: 50000, Currency: "USD"}
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrPaymentUnavailable) {
		t.Fatalf("mismatched currency err = %v", err)
	}

	svc, d = newTestService(50000)
	d.payments.err = errFake
	if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, errFake) {
		t.Fatalf("payment prepare error err = %v", err)
	}

	svc, d = newTestService(50000)
	b, p, err := svc.CreateBooking(ctx, validInput())
	if err != nil {
		t.Fatalf("paid CreateBooking: %v", err)
	}
	if d.store.called("CreateWithPayment") != 1 || d.store.called("Create") != 0 {
		t.Fatalf("calls = %v", d.store.calls)
	}
	if b.Status != models.BookingStatusPending || p == nil || p.Amount != 50000 || p.UserID != testUserID {
		t.Fatalf("booking=%+v payment=%+v", b, p)
	}
	if d.payments.last.Amount != 50000 || d.payments.last.Currency != "IDR" || d.payments.last.ServiceID != testServiceID {
		t.Fatalf("payment request = %+v", d.payments.last)
	}
	if d.notifier.confirmations != 0 {
		t.Fatal("paid booking must not notify before payment settles")
	}
}

func seedBooking(status string) models.Booking {
	in := validInput()
	return models.Booking{
		ID:            1,
		UserID:        testUserID,
		ProviderID:    testProviderID,
		ServiceID:     in.ServiceID,
		BookingDate:   in.BookingDate,
		StartTime:     in.StartTime,
		EndTime:       in.EndTime,
		Status:        status,
		PaymentStatus: models.BookingPaymentNotRequired,
	}
}

func TestCancelBooking(t *testing.T) {
	ctx := context.Background()

	svc, _ := newTestService(0)
	if err := svc.CancelBooking(ctx, 99, testUserID, "r"); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("not found err = %v", err)
	}

	svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
	if err := svc.CancelBooking(ctx, 1, testUserID+1, "r"); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("other user err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusCancelled))
	if err := svc.CancelBooking(ctx, 1, testUserID, "r"); err != nil {
		t.Fatalf("already cancelled err = %v", err)
	}
	if d.store.called("UpdateStatus") != 0 || d.notifier.cancellations != 0 {
		t.Fatal("already cancelled must be a no-op")
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusCompleted))
	if err := svc.CancelBooking(ctx, 1, testUserID, "r"); err == nil {
		t.Fatal("completed booking must not be cancellable")
	}
	if d.store.called("UpdateStatus") != 0 {
		t.Fatal("UpdateStatus must not be called for completed booking")
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.store.updateStatusErr = errFake
	if err := svc.CancelBooking(ctx, 1, testUserID, "r"); !errors.Is(err, errFake) {
		t.Fatalf("UpdateStatus error err = %v", err)
	}
	if d.notifier.cancellations != 0 {
		t.Fatal("failed cancel must not notify")
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	if err := svc.CancelBooking(ctx, 1, testUserID, "changed plans"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if d.store.lastStatus != models.BookingStatusCancelled || d.store.lastReason != "changed plans" {
		t.Fatalf("UpdateStatus(%q, %q)", d.store.lastStatus, d.store.lastReason)
	}
	if d.notifier.cancellations != 1 {
		t.Fatalf("cancellations = %d", d.notifier.cancellations)
	}
}

func TestRescheduleBooking(t *testing.T) {
	ctx := context.Background()
	newDate := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	newStart := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)
	newEnd := newStart.Add(time.Hour)

	svc, _ := newTestService(0)
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("not found err = %v", err)
	}

	for _, status := range []string{models.BookingStatusCancelled, models.BookingStatusCompleted} {
		svc, d := newTestService(0, seedBooking(status))
		if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, ErrNotReschedulable) {
			t.Fatalf("%s booking err = %v, want ErrNotReschedulable", status, err)
		}
		if d.store.called("Reschedule") != 0 {
			t.Fatalf("%s: Reschedule called", status)
		}
	}

	svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.avail.conflicts = []availability.BookingConflict{{}}
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("availability conflict err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.avail.err = errFake
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, errFake) {
		t.Fatalf("availability error err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.store.conflict = true
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("repo conflict err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.store.checkConflictErr = errFake
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, errFake) {
		t.Fatalf("CheckConflict error err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.store.rescheduleErr = sqlStateErr{"23P01"}
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("23P01 on Reschedule err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.store.rescheduleErr = errFake
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); !errors.Is(err, errFake) {
		t.Fatalf("Reschedule error err = %v", err)
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusPending))
	if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newEnd); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	got := d.store.get(1)
	if !got.BookingDate.Equal(newDate) || !got.StartTime.Equal(newStart) || !got.EndTime.Equal(newEnd) {
		t.Fatalf("booking not updated: %+v", got)
	}
}

func TestRescheduleBookingEnforcesServiceDuration(t *testing.T) {
	ctx := context.Background()
	newDate := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	newStart := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		end  time.Time
		want error
	}{
		{"shorter than service duration", newStart.Add(30 * time.Minute), ErrInvalidInput},
		{"longer than service duration", newStart.Add(2 * time.Hour), ErrInvalidInput},
		{"end before start", newStart.Add(-time.Hour), ErrInvalidInput},
		{"exact duration", newStart.Add(time.Hour), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
			err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, tc.end)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			wantCalls := 0
			if tc.want == nil {
				wantCalls = 1
			}
			if d.store.called("Reschedule") != wantCalls {
				t.Fatalf("Reschedule calls = %d, want %d", d.store.called("Reschedule"), wantCalls)
			}
			if d.avail.calls != wantCalls {
				t.Fatalf("availability checks = %d: validation must run before availability", d.avail.calls)
			}
		})
	}

	t.Run("service lookup failure", func(t *testing.T) {
		svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
		d.catalog.services = map[string]*models.Service{}
		err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newStart.Add(time.Hour))
		if !errors.Is(err, ErrServiceNotFound) {
			t.Fatalf("err = %v, want ErrServiceNotFound", err)
		}
	})
	t.Run("non-positive service duration", func(t *testing.T) {
		svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
		d.catalog.services[testServiceID].Duration = 0
		err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newStart)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("booking without a service only needs a valid range", func(t *testing.T) {
		seed := seedBooking(models.BookingStatusConfirmed)
		seed.ServiceID = nil
		svc, _ := newTestService(0, seed)
		if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newStart.Add(90*time.Minute)); err != nil {
			t.Fatalf("err = %v", err)
		}
		if err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newStart); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("empty range err = %v, want ErrInvalidInput", err)
		}
	})
}

// A booking cancelled between the load and the write must not be revived or
// moved: the guarded write reports ErrNotReschedulable.
func TestRescheduleBookingStatusChangedAfterLoad(t *testing.T) {
	ctx := context.Background()
	newDate := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	newStart := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)
	svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
	d.avail.onCheck = func() { d.store.bookings[1].Status = models.BookingStatusCancelled }
	err := svc.RescheduleBooking(ctx, 1, testUserID, newDate, newStart, newStart.Add(time.Hour))
	if !errors.Is(err, ErrNotReschedulable) {
		t.Fatalf("err = %v, want ErrNotReschedulable", err)
	}
	if got := d.store.get(1); got.Status != models.BookingStatusCancelled || got.StartTime.Equal(newStart) {
		t.Fatalf("booking changed: %+v", got)
	}
}

func TestConfirmBooking(t *testing.T) {
	ctx := context.Background()

	svc, _ := newTestService(0)
	if err := svc.ConfirmBooking(ctx, 1); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("not found err = %v", err)
	}

	svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
	if err := svc.ConfirmBooking(ctx, 1); err == nil {
		t.Fatal("non-pending booking must not be confirmable")
	}

	unpaid := seedBooking(models.BookingStatusPending)
	pid := uint(3)
	unpaid.PaymentID = &pid
	unpaid.PaymentStatus = models.BookingPaymentPending
	svc, d = newTestService(0, unpaid)
	if err := svc.ConfirmBooking(ctx, 1); err == nil {
		t.Fatal("unpaid booking must not be confirmable")
	}
	if d.store.called("UpdateStatus") != 0 {
		t.Fatal("UpdateStatus called for unpaid booking")
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusPending))
	d.store.updateStatusErr = errFake
	if err := svc.ConfirmBooking(ctx, 1); !errors.Is(err, errFake) {
		t.Fatalf("UpdateStatus error err = %v", err)
	}

	paid := unpaid
	paid.PaymentStatus = models.BookingPaymentPaid
	svc, d = newTestService(0, paid)
	if err := svc.ConfirmBooking(ctx, 1); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if d.store.get(1).Status != models.BookingStatusConfirmed || d.notifier.confirmations != 1 {
		t.Fatalf("status=%s confirmations=%d", d.store.get(1).Status, d.notifier.confirmations)
	}
}

func TestCompleteBooking(t *testing.T) {
	ctx := context.Background()

	svc, _ := newTestService(0)
	if err := svc.CompleteBooking(ctx, 1); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("not found err = %v", err)
	}

	svc, _ = newTestService(0, seedBooking(models.BookingStatusPending))
	if err := svc.CompleteBooking(ctx, 1); err == nil {
		t.Fatal("unconfirmed booking must not be completable")
	}

	unpaid := seedBooking(models.BookingStatusConfirmed)
	pid := uint(3)
	unpaid.PaymentID = &pid
	unpaid.PaymentStatus = models.BookingPaymentPending
	svc, d := newTestService(0, unpaid)
	if err := svc.CompleteBooking(ctx, 1); err == nil {
		t.Fatal("unpaid booking must not be completable")
	}
	if d.store.called("UpdateStatus") != 0 {
		t.Fatal("UpdateStatus called for unpaid booking")
	}

	svc, d = newTestService(0, seedBooking(models.BookingStatusConfirmed))
	if err := svc.CompleteBooking(ctx, 1); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if d.store.get(1).Status != models.BookingStatusCompleted {
		t.Fatalf("status = %s", d.store.get(1).Status)
	}
}

func TestListUserBookingsClampsPagination(t *testing.T) {
	ctx := context.Background()
	cases := []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, 0, 20, 0},
		{500, 0, 100, 0},
		{10, -5, 10, 0},
	}
	for _, tc := range cases {
		svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
		if _, err := svc.ListUserBookings(ctx, testUserID, tc.limit, tc.offset); err != nil {
			t.Fatalf("ListUserBookings: %v", err)
		}
		if d.store.lastListLimit != tc.wantLimit || d.store.lastListOffset != tc.wantOffset {
			t.Errorf("(%d,%d) -> (%d,%d), want (%d,%d)", tc.limit, tc.offset,
				d.store.lastListLimit, d.store.lastListOffset, tc.wantLimit, tc.wantOffset)
		}
	}
}

func TestGetBookingByIDAndStats(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(0, seedBooking(models.BookingStatusConfirmed))
	if _, err := svc.GetBookingByID(ctx, 1, testUserID+1); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("other user err = %v", err)
	}
	d.store.stats = &models.ProviderStats{}
	if _, err := svc.GetProviderStats(ctx, testProviderID); err != nil {
		t.Fatalf("GetProviderStats: %v", err)
	}
	d.store.statsErr = errFake
	if _, err := svc.GetProviderStats(ctx, testProviderID); !errors.Is(err, errFake) {
		t.Fatalf("stats err = %v", err)
	}
}

func TestSlotInvalidationOnMutations(t *testing.T) {
	ctx := context.Background()
	in := validInput()
	newDate := in.BookingDate.AddDate(0, 0, 7)

	paid := seedBooking(models.BookingStatusPending)
	pid := uint(9)
	paid.PaymentID = &pid
	paid.PaymentStatus = models.BookingPaymentPaid

	tests := []struct {
		name  string
		price float64
		seed  []models.Booking
		run   func(*Service) error
	}{
		{"create free", 0, nil, func(s *Service) error { _, _, err := s.CreateBooking(ctx, validInput()); return err }},
		{"create paid", 50000, nil, func(s *Service) error { _, _, err := s.CreateBooking(ctx, validInput()); return err }},
		{"cancel", 0, []models.Booking{seedBooking(models.BookingStatusConfirmed)}, func(s *Service) error { return s.CancelBooking(ctx, 1, testUserID, "r") }},
		{"reschedule", 0, []models.Booking{seedBooking(models.BookingStatusConfirmed)}, func(s *Service) error {
			return s.RescheduleBooking(ctx, 1, testUserID, newDate, in.StartTime, in.EndTime)
		}},
		{"confirm", 0, []models.Booking{paid}, func(s *Service) error { return s.ConfirmBooking(ctx, 1) }},
		{"complete", 0, []models.Booking{seedBooking(models.BookingStatusConfirmed)}, func(s *Service) error { return s.CompleteBooking(ctx, 1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(tt.price, tt.seed...)
			inv := &fakeInvalidator{}
			if got := svc.WithSlotInvalidator(inv); got != svc {
				t.Fatal("WithSlotInvalidator must return the same service")
			}
			if err := tt.run(svc); err != nil {
				t.Fatalf("mutation failed: %v", err)
			}
			if len(inv.providers) != 1 || inv.providers[0] != testProviderID {
				t.Fatalf("invalidated %v, want [%s]", inv.providers, testProviderID)
			}
		})
	}
}

func TestSlotInvalidationSkippedWhenNothingChanged(t *testing.T) {
	ctx := context.Background()

	t.Run("failed create", func(t *testing.T) {
		svc, d := newTestService(0)
		d.store.conflict = true
		inv := &fakeInvalidator{}
		svc.WithSlotInvalidator(inv)
		if _, _, err := svc.CreateBooking(ctx, validInput()); !errors.Is(err, ErrBookingConflict) {
			t.Fatalf("err = %v", err)
		}
		if len(inv.providers) != 0 {
			t.Fatalf("invalidated on failed create: %v", inv.providers)
		}
	})
	t.Run("store error on create", func(t *testing.T) {
		svc, d := newTestService(0)
		d.store.createErr = errors.New("db down")
		inv := &fakeInvalidator{}
		svc.WithSlotInvalidator(inv)
		if _, _, err := svc.CreateBooking(ctx, validInput()); err == nil {
			t.Fatal("expected error")
		}
		if len(inv.providers) != 0 {
			t.Fatalf("invalidated on failed create: %v", inv.providers)
		}
	})
	t.Run("cancel already cancelled", func(t *testing.T) {
		svc, _ := newTestService(0, seedBooking(models.BookingStatusCancelled))
		inv := &fakeInvalidator{}
		svc.WithSlotInvalidator(inv)
		if err := svc.CancelBooking(ctx, 1, testUserID, "r"); err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(inv.providers) != 0 {
			t.Fatalf("invalidated without a status change: %v", inv.providers)
		}
	})
	t.Run("nil invalidator is safe", func(t *testing.T) {
		svc, _ := newTestService(0)
		if _, _, err := svc.CreateBooking(ctx, validInput()); err != nil {
			t.Fatalf("err = %v", err)
		}
	})
}

// A recording invalidator cannot influence the mutation's result: the
// interface returns nothing, so the booking is created exactly as without one.
func TestSlotInvalidatorCannotChangeMutationResult(t *testing.T) {
	ctx := context.Background()
	plain, _ := newTestService(0)
	want, _, err := plain.CreateBooking(ctx, validInput())
	if err != nil {
		t.Fatal(err)
	}
	withInv, _ := newTestService(0)
	withInv.WithSlotInvalidator(&fakeInvalidator{})
	got, _, err := withInv.CreateBooking(ctx, validInput())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Status != want.Status || got.PaymentStatus != want.PaymentStatus || got.ProviderID != want.ProviderID {
		t.Fatalf("result changed: got %+v want %+v", got, want)
	}
}
