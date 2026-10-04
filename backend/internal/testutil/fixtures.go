package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"temu.in/backend/internal/models"
)

var fixtureSeq atomic.Uint64

func nextSeq() uint64 { return fixtureSeq.Add(1) }

// CreateUser inserts a verified user with a unique email and the given role
// ("user", "provider", "admin").
func CreateUser(t *testing.T, db *gorm.DB, role string) *models.User {
	t.Helper()
	n := nextSeq()
	u := &models.User{
		Email:      fmt.Sprintf("user%d@example.test", n),
		Password:   "not-a-real-hash",
		Name:       fmt.Sprintf("User %d", n),
		Role:       role,
		IsVerified: true,
	}
	if err := db.Omit(clause.Associations).Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// CreateProvider inserts an active provider owned by userID.
func CreateProvider(t *testing.T, db *gorm.DB, userID uint) *models.ServiceProvider {
	t.Helper()
	n := nextSeq()
	p := &models.ServiceProvider{
		UserID:       userID,
		BusinessName: fmt.Sprintf("Provider %d", n),
		BusinessType: "salon",
		Description:  "test provider",
		Address:      "Jl. Test No. 1",
		PricingInfo:  "{}",
		IsActive:     true,
	}
	if err := db.Omit(clause.Associations).Create(p).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	return p
}

// CreateService inserts an active service for providerID.
func CreateService(t *testing.T, db *gorm.DB, providerID string, durationMinutes int, price float64) *models.Service {
	t.Helper()
	n := nextSeq()
	s := &models.Service{
		ProviderID: providerID,
		Name:       fmt.Sprintf("Service %d", n),
		Duration:   durationMinutes,
		Price:      price,
		IsActive:   true,
	}
	if err := db.Omit(clause.Associations).Create(s).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	return s
}

// CreateSchedule inserts a weekly availability window ("HH:MM" times).
func CreateSchedule(t *testing.T, db *gorm.DB, providerID string, dayOfWeek int, start, end string) *models.AvailabilitySchedule {
	t.Helper()
	s := &models.AvailabilitySchedule{
		ProviderID:  providerID,
		DayOfWeek:   dayOfWeek,
		StartTime:   start,
		EndTime:     end,
		IsAvailable: true,
	}
	if err := db.Omit(clause.Associations).Create(s).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	return s
}

// MustClock parses an "HH:MM" clock string into the time.Time form the
// booking model stores in its start_time/end_time columns.
func MustClock(t *testing.T, hhmm string) time.Time {
	t.Helper()
	v, err := time.Parse("15:04", hhmm)
	if err != nil {
		t.Fatalf("parse clock %q: %v", hhmm, err)
	}
	return v
}

// MustDate parses a "YYYY-MM-DD" date.
func MustDate(t *testing.T, ymd string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		t.Fatalf("parse date %q: %v", ymd, err)
	}
	return v
}

// CreateBooking inserts a booking for userID at providerID on date
// ("YYYY-MM-DD") from start to end ("HH:MM") with the given status.
// serviceID may be nil.
func CreateBooking(t *testing.T, db *gorm.DB, userID uint, providerID string, serviceID *string, date, start, end, status string) *models.Booking {
	t.Helper()
	paymentStatus := models.BookingPaymentNotRequired
	if status == models.BookingStatusPending {
		paymentStatus = models.BookingPaymentUnpaid
	}
	b := &models.Booking{
		UserID:        userID,
		ProviderID:    providerID,
		ServiceID:     serviceID,
		CustomerName:  fmt.Sprintf("Customer %d", nextSeq()),
		BookingDate:   MustDate(t, date),
		StartTime:     MustClock(t, start),
		EndTime:       MustClock(t, end),
		Status:        status,
		PaymentStatus: paymentStatus,
	}
	if err := db.Omit(clause.Associations).Create(b).Error; err != nil {
		t.Fatalf("create booking: %v", err)
	}
	return b
}
