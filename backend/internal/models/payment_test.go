package models

import "testing"

func TestCanTransitionPaymentStatus(t *testing.T) {
	states := []string{
		PaymentStatusCreated, PaymentStatusPending, PaymentStatusSucceeded, PaymentStatusFailed,
		PaymentStatusExpired, PaymentStatusCancelled, PaymentStatusRefunded, "unknown", "",
	}
	terminalFromOpen := map[string]bool{
		PaymentStatusSucceeded: true, PaymentStatusFailed: true,
		PaymentStatusExpired: true, PaymentStatusCancelled: true,
	}
	allowed := map[string]map[string]bool{
		PaymentStatusCreated:   {PaymentStatusPending: true},
		PaymentStatusPending:   {},
		PaymentStatusSucceeded: {PaymentStatusRefunded: true},
	}
	for k := range terminalFromOpen {
		allowed[PaymentStatusCreated][k] = true
		allowed[PaymentStatusPending][k] = true
	}
	for _, from := range states {
		for _, to := range states {
			want := from == to || allowed[from][to]
			if got := CanTransitionPaymentStatus(from, to); got != want {
				t.Errorf("CanTransitionPaymentStatus(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}
