package review

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/auth"
	"temu.in/backend/internal/models"
)

type fakeReviewService struct {
	reviews []models.Review
	err     error
}

func (f *fakeReviewService) CreateReview(userID uint, bookingID uint, req CreateReviewRequest) (*models.Review, error) {
	if f.err != nil {
		return nil, f.err
	}
	r := models.Review{ID: "rev-new", BookingID: bookingID, ProviderID: "prov-1", UserID: userID, Rating: req.Rating, Comment: req.Comment, CreatedAt: time.Now()}
	return &r, nil
}

func (f *fakeReviewService) ListProviderReviews(providerID string, limit, offset int) ([]models.Review, error) {
	return f.reviews, f.err
}

func (f *fakeReviewService) ListUserReviews(userID uint, limit, offset int) ([]models.Review, error) {
	return f.reviews, f.err
}

func (f *fakeReviewService) GetProviderStats(providerID string) (*ProviderReviewStats, error) {
	return &ProviderReviewStats{AverageRating: 4.5, TotalReviews: 2}, f.err
}

func reviewFixture() *fakeReviewService {
	reviewer := models.User{ID: 20, Email: "ana@example.test", Name: "Ana", Role: "user", IsVerified: true}
	owner := models.User{ID: 7, Email: "owner@example.test", Name: "Siti", Role: "provider"}
	return &fakeReviewService{reviews: []models.Review{{
		ID: "rev-1", BookingID: 1, ProviderID: "prov-1", UserID: 20, Rating: 5, Comment: "great",
		User:     reviewer,
		Provider: models.ServiceProvider{ID: "prov-1", UserID: 7, User: owner, BusinessName: "Salon"},
		Booking:  models.Booking{ID: 1, UserID: 20, User: reviewer, AdminNotes: "secret"},
	}}}
}

func engineFor(svc reviewService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := &Handlers{service: svc}
	api := engine.Group("/api")
	h.Register(api)
	protected := api.Group("/")
	protected.Use(func(c *gin.Context) {
		if id := c.GetHeader("X-Test-User"); id != "" {
			uid, _ := strconv.Atoi(id)
			c.Set(auth.UserContextKey, &auth.Claims{UserID: uint(uid), Role: "user"})
		}
		c.Next()
	})
	h.RegisterProtected(protected)
	return engine
}

var piiKeys = map[string]bool{"email": true, "role": true, "isVerified": true, "password": true, "verification_token": true, "user": true, "admin_notes": true}

func piiIn(v interface{}) []string {
	var found []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if piiKeys[k] {
				found = append(found, k)
			}
			found = append(found, piiIn(child)...)
		}
	case []interface{}:
		for _, child := range val {
			found = append(found, piiIn(child)...)
		}
	}
	return found
}

func TestReviewResponsesExposeNoPII(t *testing.T) {
	engine := engineFor(reviewFixture())
	cases := []struct {
		name, method, path, body, user string
		status                         int
	}{
		{"provider reviews", http.MethodGet, "/api/providers/prov-1/reviews", "", "", http.StatusOK},
		{"my reviews", http.MethodGet, "/api/reviews/me", "", "20", http.StatusOK},
		{"create", http.MethodPost, "/api/bookings/1/reviews", `{"rating":4,"comment":"ok"}`, "20", http.StatusCreated},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		if tc.user != "" {
			req.Header.Set("X-Test-User", tc.user)
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, res.Code, tc.status, res.Body.String())
			continue
		}
		var body interface{}
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if bad := piiIn(body); len(bad) > 0 {
			t.Errorf("%s: response contains %v: %s", tc.name, bad, res.Body.String())
		}
	}
}

func TestProviderReviewsShape(t *testing.T) {
	engine := engineFor(reviewFixture())
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/providers/prov-1/reviews", nil))
	var body struct {
		Reviews []map[string]interface{} `json:"reviews"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || len(body.Reviews) != 1 {
		t.Fatalf("body %s err %v", res.Body.String(), err)
	}
	r := body.Reviews[0]
	if r["reviewer_name"] != "Ana" || r["rating"].(float64) != 5 || r["booking_id"].(float64) != 1 || r["provider_id"] != "prov-1" {
		t.Errorf("review = %v", r)
	}
}

func TestReviewHandlerErrors(t *testing.T) {
	engine := engineFor(&fakeReviewService{err: errors.New("boom")})
	cases := []struct {
		method, path, body, user string
		status                   int
	}{
		{http.MethodGet, "/api/providers/prov-1/reviews", "", "", http.StatusInternalServerError},
		{http.MethodGet, "/api/reviews/me", "", "20", http.StatusInternalServerError},
		{http.MethodGet, "/api/reviews/me", "", "", http.StatusUnauthorized},
		{http.MethodPost, "/api/bookings/x/reviews", `{"rating":4}`, "20", http.StatusBadRequest},
		{http.MethodPost, "/api/bookings/1/reviews", `{"rating":9}`, "20", http.StatusBadRequest},
		{http.MethodPost, "/api/bookings/1/reviews", `{"rating":4}`, "20", http.StatusBadRequest},
		{http.MethodGet, "/api/providers/prov-1/reviews/stats", "", "", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		if tc.user != "" {
			req.Header.Set("X-Test-User", tc.user)
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Errorf("%s %s: status %d, want %d", tc.method, tc.path, res.Code, tc.status)
		}
	}
}
