package booking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/payment"
)

// Compile-time checks: the real repository and the fake both satisfy bookingStore.
var (
	_ bookingStore = (*Repository)(nil)
	_ bookingStore = (*fakeStore)(nil)
)

var errFake = errors.New("fake failure")

// fakeStore is an in-memory bookingStore. It records every call by method
// name and supports per-method error injection plus a forced conflict result.
type fakeStore struct {
	mu       sync.Mutex
	bookings map[uint]*models.Booking
	payments []*models.Payment
	nextID   uint
	calls    []string

	conflict bool
	stats    *models.ProviderStats

	createErr            error
	createWithPaymentErr error
	rescheduleErr        error
	findErr              error
	listErr              error
	updateStatusErr      error
	checkConflictErr     error
	statsErr             error

	providerNames map[string]string
	serviceNames  map[string]string

	lastListLimit  int
	lastListOffset int
	lastFilter     string
	lastStatus     string
	lastReason     string
}

func newFakeStore(seed ...models.Booking) *fakeStore {
	f := &fakeStore{bookings: map[uint]*models.Booking{}}
	for i := range seed {
		b := seed[i]
		if b.ID == 0 {
			f.nextID++
			b.ID = f.nextID
		} else if b.ID > f.nextID {
			f.nextID = b.ID
		}
		f.bookings[b.ID] = &b
	}
	return f
}

func (f *fakeStore) record(name string) {
	f.calls = append(f.calls, name)
}

// called reports how many times a method was invoked.
func (f *fakeStore) called(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == name {
			n++
		}
	}
	return n
}

// get returns a copy of the stored booking, or nil.
func (f *fakeStore) get(id uint) *models.Booking {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.bookings[id]
	if !ok {
		return nil
	}
	cp := *b
	return &cp
}

func (f *fakeStore) insert(b *models.Booking) {
	f.nextID++
	b.ID = f.nextID
	now := time.Now()
	b.CreatedAt, b.UpdatedAt = now, now
	cp := *b
	f.bookings[b.ID] = &cp
}

func (f *fakeStore) Create(b *models.Booking) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("Create")
	if f.createErr != nil {
		return f.createErr
	}
	f.insert(b)
	return nil
}

func (f *fakeStore) CreateWithPayment(b *models.Booking, p *models.Payment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("CreateWithPayment")
	if f.createWithPaymentErr != nil {
		return f.createWithPaymentErr
	}
	if p == nil {
		return errors.New("payment is required")
	}
	p.ID = uint(len(f.payments) + 1)
	f.payments = append(f.payments, p)
	b.Status = models.BookingStatusPending
	b.PaymentStatus = models.BookingPaymentPending
	pid := p.ID
	b.PaymentID = &pid
	f.insert(b)
	return nil
}

// Reschedule mirrors the repository's guarded, column-scoped write: only the
// date and time columns change, and a missing, foreign, cancelled or
// completed booking yields ErrNotReschedulable.
func (f *fakeStore) Reschedule(id, userID uint, date, start, end time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("Reschedule")
	if f.rescheduleErr != nil {
		return f.rescheduleErr
	}
	b, ok := f.bookings[id]
	if !ok || b.UserID != userID || b.Status == models.BookingStatusCancelled || b.Status == models.BookingStatusCompleted {
		return ErrNotReschedulable
	}
	b.BookingDate, b.StartTime, b.EndTime = date, start, end
	b.UpdatedAt = time.Now()
	return nil
}

func (f *fakeStore) FindByID(id uint) (*models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("FindByID")
	if f.findErr != nil {
		return nil, f.findErr
	}
	b, ok := f.bookings[id]
	if !ok {
		return nil, errors.New("record not found")
	}
	cp := *b
	return &cp, nil
}

func (f *fakeStore) FindByIDForUser(id uint, userID uint) (*models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("FindByIDForUser")
	if f.findErr != nil {
		return nil, f.findErr
	}
	b, ok := f.bookings[id]
	if !ok || b.UserID != userID {
		return nil, errors.New("record not found")
	}
	cp := *b
	return &cp, nil
}

func (f *fakeStore) ListByUser(userID uint, limit, offset int) ([]models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("ListByUser")
	f.lastListLimit, f.lastListOffset = limit, offset
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []models.Booking
	for _, b := range f.bookings {
		if b.UserID == userID {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateStatus(id uint, status string, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("UpdateStatus")
	f.lastStatus, f.lastReason = status, reason
	if f.updateStatusErr != nil {
		return f.updateStatusErr
	}
	b, ok := f.bookings[id]
	if !ok {
		return errors.New("record not found")
	}
	b.Status = status
	if reason != "" {
		b.CancellationReason = reason
	}
	b.UpdatedAt = time.Now()
	return nil
}

func (f *fakeStore) CheckConflict(providerID string, date time.Time, startTime, endTime time.Time, excludeID uint) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("CheckConflict")
	if f.checkConflictErr != nil {
		return false, f.checkConflictErr
	}
	return f.conflict, nil
}

func (f *fakeStore) GetProviderStats(providerID string) (*models.ProviderStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("GetProviderStats")
	if f.statsErr != nil {
		return nil, f.statsErr
	}
	if f.stats == nil {
		return &models.ProviderStats{}, nil
	}
	cp := *f.stats
	return &cp, nil
}

// fakeAvailability returns preconfigured conflicts or an error.
type fakeAvailability struct {
	conflicts []availability.BookingConflict
	err       error
	calls     int
	onCheck   func() // optional hook, runs on every check
}

func (f *fakeAvailability) CheckBookingConflicts(providerID string, date time.Time, startTime, endTime string, excludeBookingID ...string) ([]availability.BookingConflict, error) {
	f.calls++
	if f.onCheck != nil {
		f.onCheck()
	}
	return f.conflicts, f.err
}

// fakeCatalog serves services by id.
type fakeCatalog struct {
	services map[string]*models.Service
	err      error
}

func (f *fakeCatalog) GetService(id string) (*models.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.services[id]
	if !ok {
		return nil, errors.New("service not found")
	}
	return s, nil
}

// fakeProviderSvc serves providers by id; it also satisfies auth.ProviderLookup.
type fakeProviderSvc struct {
	providers map[string]*models.ServiceProvider
	err       error
}

func (f *fakeProviderSvc) GetProvider(id string) (*models.ServiceProvider, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.providers[id]
	if !ok {
		return nil, errors.New("provider not found")
	}
	return p, nil
}

// fakeUserRepo serves users by id.
type fakeUserRepo struct {
	users map[uint]*models.User
	err   error
}

func (f *fakeUserRepo) FindByID(id uint) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	u, ok := f.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

// fakeNotifier counts notification calls.
type fakeNotifier struct {
	mu            sync.Mutex
	confirmations int
	cancellations int
	reminders     int
	err           error
}

func (f *fakeNotifier) SendBookingConfirmation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.confirmations++
	return f.err
}

func (f *fakeNotifier) SendBookingCancellation(ctx context.Context, booking *models.Booking, user *models.User, provider *models.ServiceProvider) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancellations++
	return f.err
}

func (f *fakeNotifier) SendBookingReminder(ctx context.Context, booking *models.Booking, user *models.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reminders++
	return f.err
}

// fakePaymentCreator returns a payment built from the request unless an
// override or error is configured.
type fakePaymentCreator struct {
	payment *models.Payment
	err     error
	calls   int
	last    payment.CreateRequest
}

func (f *fakePaymentCreator) Prepare(ctx context.Context, req payment.CreateRequest) (*models.Payment, error) {
	f.calls++
	f.last = req
	if f.err != nil {
		return nil, f.err
	}
	if f.payment != nil {
		cp := *f.payment
		return &cp, nil
	}
	return &models.Payment{
		Provider:     "fake",
		OrderID:      "order-1",
		ClientSecret: "secret",
		Amount:       req.Amount,
		Currency:     req.Currency,
		Status:       models.PaymentStatusPending,
	}, nil
}

// fakeInvalidator records provider ids passed to slot-cache invalidation
// (wired by the booking-side invalidation step).
type fakeInvalidator struct {
	mu        sync.Mutex
	providers []string
}

func (f *fakeInvalidator) InvalidateProviderSlots(ctx context.Context, providerID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers = append(f.providers, providerID)
}

func TestBookingStoreImplementations(t *testing.T) {
	var stores = []bookingStore{(*Repository)(nil), newFakeStore()}
	for _, s := range stores {
		if s == nil {
			t.Fatal("nil bookingStore")
		}
	}
	svc := &Service{repo: newFakeStore(models.Booking{UserID: 1, ProviderID: "p1"})}
	b, err := svc.GetBookingByID(context.Background(), 1, 1)
	if err != nil || b.ProviderID != "p1" {
		t.Fatalf("GetBookingByID via fake = %+v, %v", b, err)
	}
}

// withNames attaches the configured provider/service names, mimicking the
// repository's preloads.
func (f *fakeStore) withNames(b models.Booking) models.Booking {
	if f.providerNames != nil {
		b.ServiceProvider.ID = b.ProviderID
		b.ServiceProvider.BusinessName = f.providerNames[b.ProviderID]
	}
	if f.serviceNames != nil && b.ServiceID != nil {
		if name, ok := f.serviceNames[*b.ServiceID]; ok {
			b.Service = &models.Service{ID: *b.ServiceID, Name: name}
		}
	}
	return b
}

func page(in []models.Booking, limit, offset int) []models.Booking {
	if offset >= len(in) {
		return []models.Booking{}
	}
	in = in[offset:]
	if limit < len(in) {
		in = in[:limit]
	}
	return in
}

func (f *fakeStore) ListByProviderWithNames(providerID, status string, limit, offset int) ([]models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("ListByProviderWithNames")
	f.lastListLimit, f.lastListOffset, f.lastFilter = limit, offset, status
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []models.Booking
	for _, b := range f.bookings {
		if b.ProviderID == providerID && (status == "" || b.Status == status) {
			out = append(out, f.withNames(*b))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].BookingDate.Equal(out[j].BookingDate) {
			return out[i].BookingDate.After(out[j].BookingDate)
		}
		return out[i].StartTime.After(out[j].StartTime)
	})
	return page(out, limit, offset), nil
}

func (f *fakeStore) ListByUserWithNames(userID uint, limit, offset int) ([]models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("ListByUserWithNames")
	f.lastListLimit, f.lastListOffset = limit, offset
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []models.Booking
	for _, b := range f.bookings {
		if b.UserID == userID {
			out = append(out, f.withNames(*b))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return page(out, limit, offset), nil
}

func (f *fakeStore) FindByIDForUserWithNames(id uint, userID uint) (*models.Booking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("FindByIDForUserWithNames")
	if f.findErr != nil {
		return nil, f.findErr
	}
	b, ok := f.bookings[id]
	if !ok || b.UserID != userID {
		return nil, errors.New("record not found")
	}
	cp := f.withNames(*b)
	return &cp, nil
}
