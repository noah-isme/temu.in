package models

import "testing"

func TestCanTransitionBookingStatus(t *testing.T) {
	states := []string{
		BookingStatusPending, BookingStatusConfirmed, BookingStatusCompleted,
		BookingStatusCancelled, BookingStatusPaymentFailed, "unknown", "",
	}
	allowed := map[string]map[string]bool{
		BookingStatusPending:       {BookingStatusConfirmed: true, BookingStatusCancelled: true, BookingStatusPaymentFailed: true},
		BookingStatusConfirmed:     {BookingStatusCompleted: true, BookingStatusCancelled: true},
		BookingStatusPaymentFailed: {BookingStatusCancelled: true},
	}
	for _, from := range states {
		for _, to := range states {
			want := from == to || allowed[from][to]
			if got := CanTransitionBookingStatus(from, to); got != want {
				t.Errorf("CanTransitionBookingStatus(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}
