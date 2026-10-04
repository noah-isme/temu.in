package models

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm/schema"
)

func TestParseClock(t *testing.T) {
	for _, in := range []string{"09:30", "09:30:00", "09:30:00.000000"} {
		got, err := ParseClock(in)
		if err != nil || got.Hour() != 9 || got.Minute() != 30 {
			t.Errorf("ParseClock(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseClock("25:00"); err == nil {
		t.Error("ParseClock(25:00) should fail")
	}
}

func TestClockSerializerRoundTrip(t *testing.T) {
	s, err := schema.Parse(&Booking{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := s.LookUpField("StartTime")
	if field == nil || field.Serializer == nil {
		t.Fatal("StartTime must use the clock serializer")
	}
	ctx := context.Background()
	ser := clockSerializer{}

	for _, dbValue := range []interface{}{"14:05:00", []byte("14:05"), time.Date(2024, 5, 1, 14, 5, 0, 0, time.UTC)} {
		var b Booking
		if err := ser.Scan(ctx, field, reflect.ValueOf(&b), dbValue); err != nil {
			t.Fatalf("Scan(%v): %v", dbValue, err)
		}
		if b.StartTime.Format("15:04") != "14:05" {
			t.Errorf("Scan(%v) = %v", dbValue, b.StartTime)
		}
	}
	var b Booking
	if err := ser.Scan(ctx, field, reflect.ValueOf(&b), nil); err != nil || !b.StartTime.IsZero() {
		t.Errorf("Scan(nil) = %v, %v", b.StartTime, err)
	}
	if err := ser.Scan(ctx, field, reflect.ValueOf(&b), "nope"); err == nil {
		t.Error("Scan(invalid) should fail")
	}
	if err := ser.Scan(ctx, field, reflect.ValueOf(&b), 42); err == nil {
		t.Error("Scan(int) should fail")
	}

	v, err := ser.Value(ctx, field, reflect.Value{}, time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil || v != "09:00:00" {
		t.Errorf("Value = %v, %v", v, err)
	}
	if v, err := ser.Value(ctx, field, reflect.Value{}, (*time.Time)(nil)); err != nil || v != nil {
		t.Errorf("Value(nil ptr) = %v, %v", v, err)
	}
	if _, err := ser.Value(ctx, field, reflect.Value{}, "x"); err == nil {
		t.Error("Value(string) should fail")
	}
}

func TestParseClockEndOfDay(t *testing.T) {
	for _, in := range []string{"24:00", "24:00:00", "24:00:00.000000"} {
		got, err := ParseClock(in)
		if err != nil || !got.Equal(EndOfDayClock) {
			t.Errorf("ParseClock(%q) = %v, %v; want EndOfDayClock", in, got, err)
		}
	}
	for _, in := range []string{"24:00:01", "24:01", "24:30:00", "25:00:00"} {
		if _, err := ParseClock(in); err == nil {
			t.Errorf("ParseClock(%q) should fail", in)
		}
	}
	// Duration arithmetic from a start time lands on the sentinel.
	if got := time.Date(0, 1, 1, 23, 0, 0, 0, time.UTC).Add(time.Hour); !got.Equal(EndOfDayClock) {
		t.Errorf("23:00 + 1h = %v, want EndOfDayClock", got)
	}
}

func TestFormatClock(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	for _, tc := range []struct {
		name string
		in   time.Time
		want string
		hm   string
	}{
		{"utc", time.Date(0, 1, 1, 9, 5, 0, 0, time.UTC), "09:05:00", "09:05"},
		{"jakarta wall clock is not shifted", time.Date(0, 1, 1, 15, 4, 5, 0, jakarta), "15:04:05", "15:04"},
		{"jakarta on a real date", time.Date(2030, 1, 7, 23, 30, 0, 0, jakarta), "23:30:00", "23:30"},
		{"fractional seconds", time.Date(0, 1, 1, 9, 0, 0, 123456000, time.UTC), "09:00:00.123456", "09:00"},
		{"end of day", EndOfDayClock, "24:00:00", "24:00"},
		{"end of day in jakarta", time.Date(0, 1, 2, 0, 0, 0, 0, jakarta), "24:00:00", "24:00"},
		{"midnight start", time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), "00:00:00", "00:00"},
	} {
		if got := FormatClock(tc.in); got != tc.want {
			t.Errorf("%s: FormatClock = %q, want %q", tc.name, got, tc.want)
		}
		if got := FormatClockHM(tc.in); got != tc.hm {
			t.Errorf("%s: FormatClockHM = %q, want %q", tc.name, got, tc.hm)
		}
	}
}

func TestClockSerializerSubSecondAndZoneRoundTrip(t *testing.T) {
	s, err := schema.Parse(&Booking{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := s.LookUpField("StartTime")
	ctx := context.Background()
	ser := clockSerializer{}

	roundTrip := func(in time.Time) time.Time {
		t.Helper()
		v, err := ser.Value(ctx, field, reflect.Value{}, in)
		if err != nil {
			t.Fatalf("Value(%v): %v", in, err)
		}
		var b Booking
		if err := ser.Scan(ctx, field, reflect.ValueOf(&b), v); err != nil {
			t.Fatalf("Scan(%v): %v", v, err)
		}
		return b.StartTime
	}

	// Postgres `time` keeps microseconds.
	in := time.Date(0, 1, 1, 9, 30, 15, 123456000, time.UTC)
	if got := roundTrip(in); !got.Equal(in) {
		t.Errorf("sub-second round trip = %v, want %v", got, in)
	}

	// Non-UTC zones must not shift the wall clock.
	if jakarta, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		wall := time.Date(0, 1, 1, 15, 4, 5, 0, jakarta)
		v, _ := ser.Value(ctx, field, reflect.Value{}, wall)
		if v != "15:04:05" {
			t.Errorf("Value(Jakarta 15:04:05) = %v, want 15:04:05", v)
		}
		if got := roundTrip(wall); got.Hour() != 15 || got.Minute() != 4 || got.Second() != 5 {
			t.Errorf("Jakarta round trip = %v", got)
		}
		pv, _ := ser.Value(ctx, field, reflect.Value{}, &wall)
		if pv != "15:04:05" {
			t.Errorf("Value(*Jakarta) = %v", pv)
		}
	}

	// 24:00:00 round-trips through Value and Scan.
	if v, err := ser.Value(ctx, field, reflect.Value{}, EndOfDayClock); err != nil || v != "24:00:00" {
		t.Errorf("Value(EndOfDayClock) = %v, %v", v, err)
	}
	for _, dbValue := range []interface{}{"24:00:00", []byte("24:00")} {
		var b Booking
		if err := ser.Scan(ctx, field, reflect.ValueOf(&b), dbValue); err != nil || !b.StartTime.Equal(EndOfDayClock) {
			t.Errorf("Scan(%v) = %v, %v", dbValue, b.StartTime, err)
		}
	}
	if got := roundTrip(EndOfDayClock); !got.Equal(EndOfDayClock) {
		t.Errorf("end-of-day round trip = %v", got)
	}
}
