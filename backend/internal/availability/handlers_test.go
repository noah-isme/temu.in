package availability

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
)

const (
	hSecret   = "availability-handler-secret"
	hOwnerID  = uint(10)
	hOtherID  = uint(11)
	hAdminID  = uint(12)
	hProvider = "p1"
)

// memRepo is an in-memory RepositoryInterface scoped by provider like the
// real repository: rows of another provider are "not found".
type memRepo struct {
	schedules  map[string]models.AvailabilitySchedule
	exceptions map[string]models.AvailabilityException
	slots      []TimeSlot
	slotsErr   error
	listErr    error
	seq        int
}

func newMemRepo() *memRepo {
	return &memRepo{schedules: map[string]models.AvailabilitySchedule{}, exceptions: map[string]models.AvailabilityException{}}
}

func (r *memRepo) nextID() string { r.seq++; return fmt.Sprintf("id-%d", r.seq) }

func (r *memRepo) CreateSchedule(s *models.AvailabilitySchedule) error {
	s.ID = r.nextID()
	r.schedules[s.ID] = *s
	return nil
}

func (r *memRepo) GetSchedulesByProvider(providerID string) ([]models.AvailabilitySchedule, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []models.AvailabilitySchedule
	for _, s := range r.schedules {
		if s.ProviderID == providerID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *memRepo) UpdateSchedule(id, providerID string, s *models.AvailabilitySchedule) error {
	if cur, ok := r.schedules[id]; !ok || cur.ProviderID != providerID {
		return gorm.ErrRecordNotFound
	}
	r.schedules[id] = *s
	return nil
}

func (r *memRepo) DeleteSchedule(id, providerID string) error {
	if cur, ok := r.schedules[id]; !ok || cur.ProviderID != providerID {
		return gorm.ErrRecordNotFound
	}
	delete(r.schedules, id)
	return nil
}

func (r *memRepo) CreateException(e *models.AvailabilityException) error {
	e.ID = r.nextID()
	r.exceptions[e.ID] = *e
	return nil
}

func (r *memRepo) GetExceptionsByProviderAndDateRange(providerID string, start, end time.Time) ([]models.AvailabilityException, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []models.AvailabilityException
	for _, e := range r.exceptions {
		if e.ProviderID == providerID && !e.Date.Before(start) && !e.Date.After(end) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *memRepo) UpdateException(id, providerID string, e *models.AvailabilityException) error {
	if cur, ok := r.exceptions[id]; !ok || cur.ProviderID != providerID {
		return gorm.ErrRecordNotFound
	}
	r.exceptions[id] = *e
	return nil
}

func (r *memRepo) DeleteException(id, providerID string) error {
	if cur, ok := r.exceptions[id]; !ok || cur.ProviderID != providerID {
		return gorm.ErrRecordNotFound
	}
	delete(r.exceptions, id)
	return nil
}

func (r *memRepo) GetAvailableSlots(string, time.Time, int) ([]TimeSlot, error) {
	return r.slots, r.slotsErr
}

type catalogMap map[string]*models.Service

func (c catalogMap) GetService(id string) (*models.Service, error) {
	if s, ok := c[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func registeredEngine(repo *memRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	providers := &fakeProviders{providers: map[string]*models.ServiceProvider{
		hProvider: {ID: hProvider, UserID: hOwnerID},
		"p2":      {ID: "p2", UserID: hOtherID},
	}}
	catalog := catalogMap{
		"s1":       {ID: "s1", ProviderID: hProvider, Duration: 60, IsActive: true},
		"inactive": {ID: "inactive", ProviderID: hProvider, Duration: 60, IsActive: false},
		"foreign":  {ID: "foreign", ProviderID: "p2", Duration: 30, IsActive: true},
	}
	engine := gin.New()
	NewHandlers(NewService(repo), providers, catalog).Register(engine.Group("/api"), hSecret)
	return engine
}

type caller struct {
	t      *testing.T
	engine *gin.Engine
}

func (c caller) as(userID uint, role, method, path, body string) (int, map[string]any) {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != 0 {
		tok, err := auth.NewToken(hSecret, userID, role, time.Hour)
		if err != nil {
			c.t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	c.engine.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestHandlerSlots(t *testing.T) {
	repo := newMemRepo()
	repo.slots = []TimeSlot{{StartTime: "09:00", EndTime: "10:00"}}
	c := caller{t, registeredEngine(repo)}

	code, body := c.as(0, "", http.MethodGet, "/api/providers/p1/availability?date=2030-01-07&serviceId=s1", "")
	if code != http.StatusOK {
		t.Fatalf("slots: %d %v", code, body)
	}
	for _, k := range []string{"date", "provider_id", "service_id", "available_slots", "slots"} {
		if _, ok := body[k]; !ok {
			t.Errorf("slots response missing %q: %v", k, body)
		}
	}
	if body["date"] != "2030-01-07" || body["provider_id"] != hProvider || body["service_id"] != "s1" {
		t.Errorf("slots body = %v", body)
	}
	if code, body := c.as(0, "", http.MethodGet, "/api/availability?providerId=p1&service_id=s1&date=2030-01-07", ""); code != http.StatusOK {
		t.Errorf("query-param form: %d %v", code, body)
	}

	cases := []struct {
		name string
		path string
		want int
	}{
		{"missing serviceId", "/api/providers/p1/availability?date=2030-01-07", http.StatusBadRequest},
		{"bad date", "/api/providers/p1/availability?date=07-01-2030&serviceId=s1", http.StatusBadRequest},
		{"unknown service", "/api/providers/p1/availability?date=2030-01-07&serviceId=nope", http.StatusNotFound},
		{"service of another provider", "/api/providers/p1/availability?date=2030-01-07&serviceId=foreign", http.StatusBadRequest},
		{"inactive service", "/api/providers/p1/availability?date=2030-01-07&serviceId=inactive", http.StatusBadRequest},
		{"missing providerId", "/api/availability?serviceId=s1", http.StatusBadRequest},
	}
	for _, tc := range cases {
		if code, body := c.as(0, "", http.MethodGet, tc.path, ""); code != tc.want {
			t.Errorf("%s: %d, want %d (%v)", tc.name, code, tc.want, body)
		}
	}

	repo.slotsErr = errors.New("db down")
	if code, _ := c.as(0, "", http.MethodGet, "/api/providers/p1/availability?date=2030-01-07&serviceId=s1", ""); code != http.StatusInternalServerError {
		t.Errorf("repo error: %d, want 500", code)
	}
}

func TestHandlerSlotsWithoutCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandlers(NewService(newMemRepo()), &fakeProviders{}).Register(engine.Group("/api"), hSecret)
	if code, _ := (caller{t, engine}).as(0, "", http.MethodGet, "/api/providers/p1/availability?serviceId=s1", ""); code != http.StatusBadRequest {
		t.Fatalf("no catalog: %d, want 400", code)
	}
}

func TestHandlerScheduleCRUD(t *testing.T) {
	repo := newMemRepo()
	c := caller{t, registeredEngine(repo)}
	base := "/api/providers/p1/schedules"
	valid := `{"day_of_week":1,"start_time":"09:00","end_time":"17:00","is_available":true}`

	if code, _ := c.as(0, "", http.MethodPost, base, valid); code != http.StatusUnauthorized {
		t.Fatalf("anonymous create: %d, want 401", code)
	}
	if code, _ := c.as(hOtherID, "provider", http.MethodPost, base, valid); code != http.StatusForbidden {
		t.Fatalf("non-owner create: %d, want 403", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPost, base, `{"day_of_week":9,"start_time":"09:00","end_time":"17:00"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid day: %d, want 400", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPost, base, `{"day_of_week":1,"start_time":"17:00","end_time":"09:00"}`); code != http.StatusBadRequest {
		t.Fatalf("end before start: %d, want 400", code)
	}

	code, body := c.as(hOwnerID, "provider", http.MethodPost, base, valid)
	if code != http.StatusCreated {
		t.Fatalf("owner create: %d %v", code, body)
	}
	id := body["schedule"].(map[string]any)["id"].(string)
	code, body = c.as(hAdminID, "admin", http.MethodPost, base, `{"day_of_week":2,"start_time":"09:00","end_time":"12:00"}`)
	if code != http.StatusCreated {
		t.Fatalf("admin create: %d, want 201", code)
	}
	if got := body["schedule"].(map[string]any)["is_available"]; got != true {
		t.Fatalf("omitted is_available: got %v, want true", got)
	}
	code, body = c.as(hOwnerID, "provider", http.MethodPost, base, `{"day_of_week":3,"start_time":"09:00","end_time":"12:00","is_available":false}`)
	if code != http.StatusCreated || body["schedule"].(map[string]any)["is_available"] != false {
		t.Fatalf("explicit is_available=false: %d %v", code, body)
	}

	code, body = c.as(hOwnerID, "provider", http.MethodGet, base, "")
	if code != http.StatusOK || len(body["schedules"].([]any)) != 3 {
		t.Fatalf("list: %d %v", code, body)
	}
	if code, _ := c.as(hOtherID, "provider", http.MethodGet, base, ""); code != http.StatusForbidden {
		t.Fatalf("non-owner list: %d, want 403", code)
	}

	update := `{"day_of_week":1,"start_time":"10:00","end_time":"18:00","is_available":true}`
	if code, body := c.as(hOwnerID, "provider", http.MethodPut, base+"/"+id, update); code != http.StatusOK || body["schedule"].(map[string]any)["start_time"] != "10:00" {
		t.Fatalf("update: %d %v", code, body)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPut, base+"/missing", update); code != http.StatusNotFound {
		t.Fatalf("update missing: %d, want 404", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPut, base+"/"+id, `{"start_time":"bad"}`); code != http.StatusBadRequest {
		t.Fatalf("update invalid: %d, want 400", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodDelete, base+"/"+id, ""); code != http.StatusOK {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodDelete, base+"/"+id, ""); code != http.StatusNotFound {
		t.Fatalf("delete again: %d, want 404", code)
	}

	repo.listErr = errors.New("db down")
	if code, _ := c.as(hOwnerID, "provider", http.MethodGet, base, ""); code != http.StatusInternalServerError {
		t.Fatalf("list error: %d, want 500", code)
	}
}

func TestHandlerExceptionCRUD(t *testing.T) {
	repo := newMemRepo()
	c := caller{t, registeredEngine(repo)}
	base := "/api/providers/p1/exceptions"
	fullDay := `{"date":"2030-01-07","is_available":false,"reason":"holiday"}`

	if code, _ := c.as(0, "", http.MethodPost, base, fullDay); code != http.StatusUnauthorized {
		t.Fatalf("anonymous create: %d, want 401", code)
	}
	if code, _ := c.as(hOtherID, "provider", http.MethodPost, base, fullDay); code != http.StatusForbidden {
		t.Fatalf("non-owner create: %d, want 403", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPost, base, `{"date":"07/01/2030"}`); code != http.StatusBadRequest {
		t.Fatalf("bad date: %d, want 400", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPost, base, `{}`); code != http.StatusBadRequest {
		t.Fatalf("missing date: %d, want 400", code)
	}

	code, body := c.as(hOwnerID, "provider", http.MethodPost, base, fullDay)
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, body)
	}
	created := body["exception"].(map[string]any)
	if created["date"] != "2030-01-07" || created["start_time"] != nil {
		t.Fatalf("created exception = %v", created)
	}
	id := created["id"].(string)

	code, body = c.as(hOwnerID, "provider", http.MethodGet, base+"?start_date=2030-01-01&end_date=2030-01-31", "")
	if code != http.StatusOK || len(body["exceptions"].([]any)) != 1 {
		t.Fatalf("list in range: %d %v", code, body)
	}
	code, body = c.as(hOwnerID, "provider", http.MethodGet, base+"?start_date=2030-02-01&end_date=2030-02-28", "")
	if code != http.StatusOK || len(body["exceptions"].([]any)) != 0 {
		t.Fatalf("list out of range: %d %v", code, body)
	}

	timed := `{"date":"2030-01-07","start_time":"12:00","end_time":"13:00","is_available":false}`
	if code, body := c.as(hOwnerID, "provider", http.MethodPut, base+"/"+id, timed); code != http.StatusOK || body["exception"].(map[string]any)["start_time"] != "12:00" {
		t.Fatalf("update: %d %v", code, body)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPut, base+"/missing", timed); code != http.StatusNotFound {
		t.Fatalf("update missing: %d, want 404", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPut, base+"/"+id, `{"date":"nope"}`); code != http.StatusBadRequest {
		t.Fatalf("update bad date: %d, want 400", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodPut, base+"/"+id, `{"date":"2030-01-07","start_time":"13:00","end_time":"12:00"}`); code != http.StatusBadRequest {
		t.Fatalf("update inverted times: %d, want 400", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodDelete, base+"/"+id, ""); code != http.StatusOK {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := c.as(hOwnerID, "provider", http.MethodDelete, base+"/"+id, ""); code != http.StatusNotFound {
		t.Fatalf("delete again: %d, want 404", code)
	}

	repo.listErr = errors.New("db down")
	if code, _ := c.as(hOwnerID, "provider", http.MethodGet, base, ""); code != http.StatusInternalServerError {
		t.Fatalf("list error: %d, want 500", code)
	}
}
