package review

import (
	"os"
	"strings"
	"testing"

	"temu.in/backend/internal/booking"
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

func TestCreateReviewIntegration(t *testing.T) {
	db := testutil.DB(t)
	svc := NewService(NewRepository(db), booking.NewRepository(db))

	owner := testutil.CreateUser(t, db, "provider")
	customer := testutil.CreateUser(t, db, "user")
	stranger := testutil.CreateUser(t, db, "user")
	p := testutil.CreateProvider(t, db, owner.ID)
	s := testutil.CreateService(t, db, p.ID, 60, 0)
	completed := testutil.CreateBooking(t, db, customer.ID, p.ID, &s.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusCompleted)
	confirmed := testutil.CreateBooking(t, db, customer.ID, p.ID, &s.ID, "2030-01-07", "11:00", "12:00", models.BookingStatusConfirmed)

	if _, err := svc.CreateReview(stranger.ID, completed.ID, CreateReviewRequest{Rating: 5}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("non-owned booking err = %v", err)
	}
	if _, err := svc.CreateReview(customer.ID, confirmed.ID, CreateReviewRequest{Rating: 5}); err == nil || !strings.Contains(err.Error(), "completed") {
		t.Fatalf("non-completed booking err = %v", err)
	}

	r, err := svc.CreateReview(customer.ID, completed.ID, CreateReviewRequest{Rating: 4, Comment: "good"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.ID == "" || r.ProviderID != p.ID || r.UserID != customer.ID || r.Rating != 4 {
		t.Fatalf("review = %+v", r)
	}
	if _, err := svc.CreateReview(customer.ID, completed.ID, CreateReviewRequest{Rating: 1}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate review err = %v", err)
	}

	second := testutil.CreateBooking(t, db, customer.ID, p.ID, &s.ID, "2030-01-08", "09:00", "10:00", models.BookingStatusCompleted)
	if _, err := svc.CreateReview(customer.ID, second.ID, CreateReviewRequest{Rating: 2}); err != nil {
		t.Fatalf("second review: %v", err)
	}

	stats, err := svc.GetProviderStats(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalReviews != 2 || stats.AverageRating != 3 {
		t.Fatalf("stats = %+v, want 2 reviews averaging 3", stats)
	}

	list, err := svc.ListProviderReviews(p.ID, 0, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListProviderReviews: %d %v", len(list), err)
	}
	if list[0].User.Name != customer.Name {
		t.Fatalf("reviewer not preloaded: %+v", list[0].User)
	}
	mine, err := svc.ListUserReviews(customer.ID, 100, 0)
	if err != nil || len(mine) != 2 || mine[0].Provider.ID != p.ID {
		t.Fatalf("ListUserReviews: %d %v", len(mine), err)
	}
	if none, err := svc.ListUserReviews(stranger.ID, 0, 0); err != nil || len(none) != 0 {
		t.Fatalf("stranger reviews: %d %v", len(none), err)
	}

	empty, err := svc.GetProviderStats(testutil.CreateProvider(t, db, owner.ID).ID)
	if err != nil || empty.TotalReviews != 0 || empty.AverageRating != 0 {
		t.Fatalf("empty stats = %+v %v", empty, err)
	}
}
