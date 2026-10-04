package availability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/provider"
	"temu.in/backend/internal/testutil"
)

func TestIsValidTime(t *testing.T) {
	cases := map[string]bool{
		"00:00": true, "09:30": true, "23:59": true,
		// Go's "15" layout accepts a one-digit hour; request binding enforces len=5.
		"9:30":  true,
		"24:00": false, "29:00": false, "09:60": false,
		"09:30:00": false, "": false, "ab:cd": false,
	}
	for in, want := range cases {
		if got := isValidTime(in); got != want {
			t.Errorf("isValidTime(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestTimesOverlap(t *testing.T) {
	cases := []struct {
		name                 string
		start1, end1, s2, e2 string
		want                 bool
	}{
		{"full-day exception (empty start)", "09:00", "10:00", "", "", true},
		{"full-day exception (empty end only)", "09:00", "10:00", "08:00", "", true},
		{"inside", "09:15", "09:45", "09:00", "10:00", true},
		{"partial overlap", "09:30", "10:30", "09:00", "10:00", true},
		{"adjacent before", "08:00", "09:00", "09:00", "10:00", false},
		{"adjacent after", "10:00", "11:00", "09:00", "10:00", false},
		{"disjoint", "12:00", "13:00", "09:00", "10:00", false},
		{"db seconds adjacent after", "10:00", "10:30", "09:30:00", "10:00:00", false},
		{"db seconds adjacent before", "09:00", "09:30", "09:30:00", "10:00:00", false},
		{"db seconds overlap", "09:45", "10:15", "09:30:00", "10:00:00", true},
	}
	for _, tc := range cases {
		if got := timesOverlap(tc.start1, tc.end1, tc.s2, tc.e2); got != tc.want {
			t.Errorf("%s: timesOverlap(%q,%q,%q,%q) = %v, want %v", tc.name, tc.start1, tc.end1, tc.s2, tc.e2, got, tc.want)
		}
	}
}

func TestTimeRangeWithin(t *testing.T) {
	cases := []struct {
		name               string
		start, end, ws, we string
		want               bool
	}{
		{"empty window start", "09:00", "10:00", "", "17:00", false},
		{"empty window end", "09:00", "10:00", "09:00", "", false},
		{"equal bounds", "09:00", "17:00", "09:00", "17:00", true},
		{"inside", "10:00", "11:00", "09:00", "17:00", true},
		{"starts before", "08:30", "09:30", "09:00", "17:00", false},
		{"ends after", "16:30", "17:30", "09:00", "17:00", false},
		{"outside", "18:00", "19:00", "09:00", "17:00", false},
		// Postgres time columns come back as HH:MM:SS; a raw string compare
		// would make "09:00:00" > "09:00" and reject the first slot of the day.
		{"db seconds at window start", "09:00", "10:00", "09:00:00", "17:00:00", true},
		{"db seconds at window end", "16:00", "17:00", "09:00:00", "17:00:00", true},
		{"db seconds outside", "17:00", "18:00", "09:00:00", "17:00:00", false},
	}
	for _, tc := range cases {
		if got := timeRangeWithin(tc.start, tc.end, tc.ws, tc.we); got != tc.want {
			t.Errorf("%s: timeRangeWithin(%q,%q,%q,%q) = %v, want %v", tc.name, tc.start, tc.end, tc.ws, tc.we, got, tc.want)
		}
	}
}

func TestClockKey(t *testing.T) {
	cases := map[string]string{"09:00": "09:00:00", "09:00:00": "09:00:00", "garbage": "garbage", "": ""}
	for in, want := range cases {
		if got := clockKey(in); got != want {
			t.Errorf("clockKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// stubRepo is a minimal RepositoryInterface for service-level tests.
type stubRepo struct {
	schedules  []models.AvailabilitySchedule
	exceptions []models.AvailabilityException
	err        error
	created    int
}

func (r *stubRepo) CreateSchedule(*models.AvailabilitySchedule) error { r.created++; return r.err }
func (r *stubRepo) GetSchedulesByProvider(string) ([]models.AvailabilitySchedule, error) {
	return r.schedules, r.err
}
func (r *stubRepo) UpdateSchedule(string, string, *models.AvailabilitySchedule) error { return r.err }
func (r *stubRepo) DeleteSchedule(string, string) error                               { return r.err }
func (r *stubRepo) CreateException(*models.AvailabilityException) error               { r.created++; return r.err }
func (r *stubRepo) GetExceptionsByProviderAndDateRange(string, time.Time, time.Time) ([]models.AvailabilityException, error) {
	return r.exceptions, r.err
}
func (r *stubRepo) UpdateException(string, string, *models.AvailabilityException) error { return r.err }
func (r *stubRepo) DeleteException(string, string) error                                { return r.err }
func (r *stubRepo) GetAvailableSlots(string, time.Time, int) ([]TimeSlot, error)        { return nil, r.err }

func TestScheduleValidation(t *testing.T) {
	cases := []struct {
		name    string
		in      models.AvailabilitySchedule
		wantErr bool
	}{
		{"valid", models.AvailabilitySchedule{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"}, false},
		{"negative day", models.AvailabilitySchedule{DayOfWeek: -1, StartTime: "09:00", EndTime: "17:00"}, true},
		{"day 7", models.AvailabilitySchedule{DayOfWeek: 7, StartTime: "09:00", EndTime: "17:00"}, true},
		{"bad time", models.AvailabilitySchedule{DayOfWeek: 1, StartTime: "9am", EndTime: "17:00"}, true},
		{"start after end", models.AvailabilitySchedule{DayOfWeek: 1, StartTime: "17:00", EndTime: "09:00"}, true},
		{"equal", models.AvailabilitySchedule{DayOfWeek: 1, StartTime: "09:00", EndTime: "09:00"}, true},
	}
	for _, tc := range cases {
		repo := &stubRepo{}
		svc := NewService(repo)
		in := tc.in
		if err := svc.CreateSchedule(&in); (err != nil) != tc.wantErr {
			t.Errorf("CreateSchedule %s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
		if err := svc.UpdateSchedule("id", "p", &in); (err != nil) != tc.wantErr {
			t.Errorf("UpdateSchedule %s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
		if tc.wantErr && repo.created != 0 {
			t.Errorf("%s: repo called despite validation error", tc.name)
		}
	}
}

func TestExceptionValidation(t *testing.T) {
	cases := []struct {
		name    string
		in      models.AvailabilityException
		wantErr bool
	}{
		{"full day", models.AvailabilityException{}, false},
		{"window", models.AvailabilityException{StartTime: "09:00", EndTime: "10:00"}, false},
		{"bad time", models.AvailabilityException{StartTime: "09:00", EndTime: "25:00"}, true},
		{"reversed", models.AvailabilityException{StartTime: "10:00", EndTime: "09:00"}, true},
		{"only start", models.AvailabilityException{StartTime: "10:00"}, true},
		{"only end", models.AvailabilityException{EndTime: "10:00"}, true},
		{"available full day is a silent no-op", models.AvailabilityException{IsAvailable: true}, true},
		{"available window", models.AvailabilityException{StartTime: "09:00", EndTime: "10:00", IsAvailable: true}, false},
		{"unavailable full day", models.AvailabilityException{IsAvailable: false}, false},
	}
	for _, tc := range cases {
		svc := NewService(&stubRepo{})
		in := tc.in
		if err := svc.CreateException(&in); (err != nil) != tc.wantErr {
			t.Errorf("CreateException %s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
		if err := svc.UpdateException("id", "p", &in); (err != nil) != tc.wantErr {
			t.Errorf("UpdateException %s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestCheckBookingConflicts(t *testing.T) {
	monday := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	schedule := []models.AvailabilitySchedule{{DayOfWeek: 1, StartTime: "09:00:00", EndTime: "17:00:00", IsAvailable: true}}

	t.Run("inside schedule from db (HH:MM:SS) at window start", func(t *testing.T) {
		svc := NewService(&stubRepo{schedules: schedule})
		conflicts, err := svc.CheckBookingConflicts("p", monday, "09:00", "10:00")
		if err != nil || len(conflicts) != 0 {
			t.Fatalf("conflicts = %+v, err %v", conflicts, err)
		}
	})
	t.Run("outside schedule", func(t *testing.T) {
		svc := NewService(&stubRepo{schedules: schedule})
		conflicts, _ := svc.CheckBookingConflicts("p", monday, "17:00", "18:00")
		if len(conflicts) != 1 || conflicts[0].Reason != "Outside scheduled availability" {
			t.Fatalf("conflicts = %+v", conflicts)
		}
	})
	t.Run("wrong weekday", func(t *testing.T) {
		svc := NewService(&stubRepo{schedules: schedule})
		conflicts, _ := svc.CheckBookingConflicts("p", monday.AddDate(0, 0, 1), "10:00", "11:00")
		if len(conflicts) != 1 {
			t.Fatalf("conflicts = %+v", conflicts)
		}
	})
	t.Run("blocked exception", func(t *testing.T) {
		svc := NewService(&stubRepo{schedules: schedule, exceptions: []models.AvailabilityException{{StartTime: "10:00:00", EndTime: "11:00:00", Reason: "lunch"}}})
		conflicts, _ := svc.CheckBookingConflicts("p", monday, "10:30", "11:30")
		if len(conflicts) != 1 || conflicts[0].Reason != "Exception: lunch" {
			t.Fatalf("conflicts = %+v", conflicts)
		}
	})
	t.Run("available exception does not block", func(t *testing.T) {
		svc := NewService(&stubRepo{schedules: schedule, exceptions: []models.AvailabilityException{{StartTime: "10:00", EndTime: "11:00", IsAvailable: true}}})
		conflicts, _ := svc.CheckBookingConflicts("p", monday, "10:00", "11:00")
		if len(conflicts) != 0 {
			t.Fatalf("conflicts = %+v", conflicts)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		svc := NewService(&stubRepo{err: errors.New("boom")})
		if _, err := svc.CheckBookingConflicts("p", monday, "10:00", "11:00"); err == nil {
			t.Fatal("expected error")
		}
	})
}

// countingRepo is a RepositoryInterface that counts slot computations and
// stores schedules/exceptions in memory.
type countingRepo struct {
	stubRepo
	slotCalls int
	slots     []TimeSlot
}

func (r *countingRepo) GetAvailableSlots(string, time.Time, int) ([]TimeSlot, error) {
	r.slotCalls++
	if r.err != nil {
		return nil, r.err
	}
	return r.slots, nil
}

func (r *countingRepo) CreateSchedule(s *models.AvailabilitySchedule) error {
	if r.err != nil {
		return r.err
	}
	s.ID = "sched-1"
	r.schedules = append(r.schedules, *s)
	return nil
}

func (r *countingRepo) CreateException(e *models.AvailabilityException) error {
	if r.err != nil {
		return r.err
	}
	e.ID = "exc-1"
	r.exceptions = append(r.exceptions, *e)
	return nil
}

var slotDate = time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)

func cachedService(t *testing.T) (*Service, *countingRepo) {
	t.Helper()
	client, _ := testutil.Redis(t)
	repo := &countingRepo{slots: []TimeSlot{{StartTime: "09:00", EndTime: "10:00"}}}
	return NewServiceWithCache(repo, NewSlotCache(client, 0)), repo
}

func TestServiceGetAvailableSlotsUsesCache(t *testing.T) {
	svc, repo := cachedService(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		slots, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60)
		if err != nil || len(slots) != 1 {
			t.Fatalf("call %d: %v, %v", i, slots, err)
		}
	}
	if repo.slotCalls != 1 {
		t.Fatalf("repo computed %d times, want 1", repo.slotCalls)
	}

	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 30); err != nil {
		t.Fatal(err)
	}
	if repo.slotCalls != 2 {
		t.Fatalf("different duration must recompute, calls = %d", repo.slotCalls)
	}
}

func TestServiceScheduleAndExceptionChangesInvalidate(t *testing.T) {
	mutations := map[string]func(*Service) error{
		"CreateSchedule": func(s *Service) error {
			return s.CreateSchedule(&models.AvailabilitySchedule{ProviderID: "p1", DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"})
		},
		"UpdateSchedule": func(s *Service) error {
			return s.UpdateSchedule("x", "p1", &models.AvailabilitySchedule{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"})
		},
		"DeleteSchedule":  func(s *Service) error { return s.DeleteSchedule("x", "p1") },
		"CreateException": func(s *Service) error { return s.CreateException(&models.AvailabilityException{ProviderID: "p1"}) },
		"UpdateException": func(s *Service) error { return s.UpdateException("x", "p1", &models.AvailabilityException{}) },
		"DeleteException": func(s *Service) error { return s.DeleteException("x", "p1") },
		"InvalidateProviderSlots": func(s *Service) error {
			s.InvalidateProviderSlots(context.Background(), "p1")
			return nil
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			svc, repo := cachedService(t)
			ctx := context.Background()
			if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
				t.Fatal(err)
			}
			if err := mutate(svc); err != nil {
				t.Fatalf("mutation: %v", err)
			}
			if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
				t.Fatal(err)
			}
			if repo.slotCalls != 2 {
				t.Fatalf("expected recompute after %s, calls = %d", name, repo.slotCalls)
			}
		})
	}
}

func TestServiceFailedMutationDoesNotInvalidate(t *testing.T) {
	svc, repo := cachedService(t)
	ctx := context.Background()
	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
		t.Fatal(err)
	}
	repo.err = errors.New("db down")
	if err := svc.DeleteSchedule("x", "p1"); err == nil {
		t.Fatal("expected error")
	}
	repo.err = nil
	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
		t.Fatal(err)
	}
	if repo.slotCalls != 1 {
		t.Fatalf("failed mutation should keep cache, calls = %d", repo.slotCalls)
	}
}

func TestServiceGetAvailableSlotsRepoErrorNotCached(t *testing.T) {
	svc, repo := cachedService(t)
	ctx := context.Background()
	repo.err = errors.New("db down")
	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err == nil {
		t.Fatal("expected error")
	}
	repo.err = nil
	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetAvailableSlots(ctx, "p1", slotDate, 60); err != nil {
		t.Fatal(err)
	}
	if repo.slotCalls != 2 {
		t.Fatalf("calls = %d, want 2 (error not cached, then one fill)", repo.slotCalls)
	}
}

func TestServiceGetAvailableSlotsRedisClosed(t *testing.T) {
	client, s := testutil.Redis(t)
	repo := &countingRepo{slots: []TimeSlot{{StartTime: "09:00", EndTime: "10:00"}}}
	cache := NewSlotCache(client, 0).WithLogger(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	svc := NewServiceWithCache(repo, cache)
	s.Close()

	slots, err := svc.GetAvailableSlots(context.Background(), "p1", slotDate, 60)
	if err != nil || len(slots) != 1 {
		t.Fatalf("closed redis must still compute: %v, %v", slots, err)
	}
	if err := svc.CreateSchedule(&models.AvailabilitySchedule{ProviderID: "p1", DayOfWeek: 1, StartTime: "09:00", EndTime: "10:00"}); err != nil {
		t.Fatalf("mutation must not fail on cache error: %v", err)
	}
}

func TestServiceWithoutCache(t *testing.T) {
	repo := &countingRepo{}
	svc := NewService(repo)
	for i := 0; i < 2; i++ {
		if _, err := svc.GetAvailableSlots(context.Background(), "p1", slotDate, 60); err != nil {
			t.Fatal(err)
		}
	}
	if repo.slotCalls != 2 {
		t.Fatalf("calls = %d", repo.slotCalls)
	}
	svc.InvalidateProviderSlots(context.Background(), "p1")
}

func TestDTONormalisation(t *testing.T) {
	cases := map[string]string{"09:00:00": "09:00", "09:00": "09:00", "17:30:59": "17:30", "junk": "junk"}
	for in, want := range cases {
		if got := clockHHMM(in); got != want {
			t.Errorf("clockHHMM(%q) = %q, want %q", in, got, want)
		}
	}

	date, _ := time.Parse(time.RFC3339, "2024-05-01T00:00:00Z")
	full := toExceptionDTO(models.AvailabilityException{ID: "e", Date: date})
	if full.Date != "2024-05-01" || full.StartTime != nil || full.EndTime != nil {
		t.Fatalf("full-day DTO = %+v", full)
	}
	raw, _ := json.Marshal(full)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if v, ok := m["start_time"]; !ok || v != nil {
		t.Fatalf("start_time must be present and null: %s", raw)
	}

	timed := toExceptionDTO(models.AvailabilityException{Date: date, StartTime: "10:00:00", EndTime: "11:30:00"})
	if *timed.StartTime != "10:00" || *timed.EndTime != "11:30" {
		t.Fatalf("timed DTO = %v %v", *timed.StartTime, *timed.EndTime)
	}

	sched := toScheduleDTO(models.AvailabilitySchedule{StartTime: "09:00:00", EndTime: "17:00:00"})
	if sched.StartTime != "09:00" || sched.EndTime != "17:00" {
		t.Fatalf("schedule DTO = %+v", sched)
	}
	if got := toScheduleDTOs(nil); got == nil || len(got) != 0 {
		t.Fatal("empty list must encode as []")
	}
	if got := toExceptionDTOs(nil); got == nil || len(got) != 0 {
		t.Fatal("empty list must encode as []")
	}
}

// fakeProviders satisfies provider.Service for authorization lookups only.
type fakeProviders struct {
	provider.Service
	providers map[string]*models.ServiceProvider
}

func (f *fakeProviders) GetProvider(id string) (*models.ServiceProvider, error) {
	if p, ok := f.providers[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

type fakeServiceCatalog struct{ svc *models.Service }

func (f fakeServiceCatalog) GetService(id string) (*models.Service, error) {
	if f.svc == nil || f.svc.ID != id {
		return nil, errors.New("not found")
	}
	return f.svc, nil
}

const handlerOwnerID = uint(5)

func handlerEngine(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	providers := &fakeProviders{providers: map[string]*models.ServiceProvider{"p1": {ID: "p1", UserID: handlerOwnerID}}}
	catalog := fakeServiceCatalog{svc: &models.Service{ID: "s1", ProviderID: "p1", Duration: 60, IsActive: true}}
	h := NewHandlers(svc, providers, catalog)
	engine := gin.New()
	api := engine.Group("/api")
	api.GET("/providers/:providerId/availability", h.GetAvailableSlots)
	protected := api.Group("/providers/:providerId")
	protected.Use(func(c *gin.Context) {
		c.Set(auth.UserContextKey, &auth.Claims{UserID: handlerOwnerID, Role: "provider"})
		c.Next()
	})
	protected.GET("/schedules", h.ListSchedules)
	protected.POST("/schedules", h.CreateSchedule)
	protected.GET("/exceptions", h.ListExceptions)
	protected.POST("/exceptions", h.CreateException)
	return engine
}

func serveBody(t *testing.T, engine *gin.Engine, method, path, body string) (int, []byte) {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

func TestHandlerSlotsIdenticalCachedVsUncached(t *testing.T) {
	slots := []TimeSlot{{StartTime: "09:00", EndTime: "10:00"}, {StartTime: "10:00", EndTime: "11:00"}}
	path := "/api/providers/p1/availability?date=2030-01-07&serviceId=s1"

	uncached := handlerEngine(NewService(&countingRepo{slots: slots}))
	codeU, bodyU := serveBody(t, uncached, http.MethodGet, path, "")

	client, _ := testutil.Redis(t)
	repo := &countingRepo{slots: slots}
	cached := handlerEngine(NewServiceWithCache(repo, NewSlotCache(client, 0)))
	codeMiss, bodyMiss := serveBody(t, cached, http.MethodGet, path, "")
	codeHit, bodyHit := serveBody(t, cached, http.MethodGet, path, "")

	if codeU != http.StatusOK || codeMiss != http.StatusOK || codeHit != http.StatusOK {
		t.Fatalf("codes = %d %d %d", codeU, codeMiss, codeHit)
	}
	if !bytes.Equal(bodyU, bodyMiss) || !bytes.Equal(bodyU, bodyHit) {
		t.Fatalf("bodies differ:\nuncached %s\nmiss     %s\nhit      %s", bodyU, bodyMiss, bodyHit)
	}
	if repo.slotCalls != 1 {
		t.Fatalf("second request should be a cache hit, calls = %d", repo.slotCalls)
	}
}

var hhmm = regexp.MustCompile(`^\d{2}:\d{2}$`)

func TestHandlerScheduleAndExceptionResponsesUseHHMM(t *testing.T) {
	repo := &countingRepo{}
	repo.schedules = []models.AvailabilitySchedule{{ID: "a", ProviderID: "p1", DayOfWeek: 1, StartTime: "09:00:00", EndTime: "17:00:00"}}
	repo.exceptions = []models.AvailabilityException{
		{ID: "b", ProviderID: "p1", Date: slotDate, StartTime: "12:00:00", EndTime: "13:00:00"},
		{ID: "c", ProviderID: "p1", Date: slotDate},
	}
	engine := handlerEngine(NewService(repo))

	code, body := serveBody(t, engine, http.MethodGet, "/api/providers/p1/schedules", "")
	var schedules struct {
		Schedules []map[string]any `json:"schedules"`
	}
	if err := json.Unmarshal(body, &schedules); err != nil || code != http.StatusOK || len(schedules.Schedules) != 1 {
		t.Fatalf("list schedules: %d %s", code, body)
	}
	for _, k := range []string{"start_time", "end_time"} {
		if v, _ := schedules.Schedules[0][k].(string); !hhmm.MatchString(v) {
			t.Errorf("schedule %s = %v", k, schedules.Schedules[0][k])
		}
	}
	if _, ok := schedules.Schedules[0]["provider"]; ok {
		t.Error("schedule DTO must not embed provider")
	}

	code, body = serveBody(t, engine, http.MethodPost, "/api/providers/p1/schedules",
		`{"day_of_week":2,"start_time":"08:00","end_time":"12:00","is_available":true}`)
	var created struct {
		Schedule map[string]any `json:"schedule"`
	}
	if err := json.Unmarshal(body, &created); err != nil || code != http.StatusCreated {
		t.Fatalf("create schedule: %d %s", code, body)
	}
	if v, _ := created.Schedule["start_time"].(string); !hhmm.MatchString(v) {
		t.Errorf("created start_time = %v", created.Schedule["start_time"])
	}

	code, body = serveBody(t, engine, http.MethodGet, "/api/providers/p1/exceptions?start_date=2030-01-01&end_date=2030-02-01", "")
	var exceptions struct {
		Exceptions []map[string]any `json:"exceptions"`
	}
	if err := json.Unmarshal(body, &exceptions); err != nil || code != http.StatusOK || len(exceptions.Exceptions) != 2 {
		t.Fatalf("list exceptions: %d %s", code, body)
	}
	if v, _ := exceptions.Exceptions[0]["start_time"].(string); !hhmm.MatchString(v) || exceptions.Exceptions[0]["date"] != "2030-01-07" {
		t.Errorf("timed exception = %v", exceptions.Exceptions[0])
	}
	if exceptions.Exceptions[1]["start_time"] != nil || exceptions.Exceptions[1]["end_time"] != nil {
		t.Errorf("full-day exception times must be null: %v", exceptions.Exceptions[1])
	}

	code, body = serveBody(t, engine, http.MethodPost, "/api/providers/p1/exceptions", `{"date":"2030-01-08","reason":"off"}`)
	var createdExc struct {
		Exception map[string]any `json:"exception"`
	}
	if err := json.Unmarshal(body, &createdExc); err != nil || code != http.StatusCreated {
		t.Fatalf("create exception: %d %s", code, body)
	}
	if createdExc.Exception["date"] != "2030-01-08" || createdExc.Exception["start_time"] != nil {
		t.Errorf("created exception = %v", createdExc.Exception)
	}
}
