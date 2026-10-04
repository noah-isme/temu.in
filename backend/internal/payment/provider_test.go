package payment

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

func TestMidtransWebhookSignature(t *testing.T) {
	provider := NewMidtransProvider("server-key", "client-key", "http://127.0.0.1")
	notification := WebhookNotification{
		OrderID:           "temu_order",
		StatusCode:        "200",
		GrossAmount:       "75000.00",
		TransactionStatus: "settlement",
	}
	digest := sha512.Sum512([]byte(notification.OrderID + notification.StatusCode + notification.GrossAmount + "server-key"))
	notification.SignatureKey = hex.EncodeToString(digest[:])

	if !provider.VerifyWebhook(notification) {
		t.Fatal("expected valid Midtrans signature")
	}
	notification.GrossAmount = "1.00"
	if provider.VerifyWebhook(notification) {
		t.Fatal("expected modified notification to fail signature verification")
	}
}

func TestNewOrderIDIsRandomAndUnique(t *testing.T) {
	first, err := NewOrderID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewOrderID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("generated duplicate order ids: %q", first)
	}
	if len(first) <= len("temu_") || len(second) <= len("temu_") {
		t.Fatalf("order ids do not contain a random suffix: %q, %q", first, second)
	}
}

func TestNormalizeWebhookStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		fraud    string
		expected string
	}{
		{name: "settled", status: "settlement", expected: "succeeded"},
		{name: "pending", status: "pending", expected: "pending"},
		{name: "expired", status: "expire", expected: "expired"},
		{name: "denied", status: "deny", expected: "failed"},
		{name: "fraud denied capture", status: "capture", fraud: "deny", expected: "failed"},
		{name: "accepted capture", status: "capture", fraud: "accept", expected: "succeeded"},
		{name: "challenged capture", status: "capture", fraud: "challenge", expected: "pending"},
		{name: "cancelled", status: "cancel", expected: "failed"},
		{name: "failure", status: "failure", expected: "failed"},
		{name: "refund", status: "refund", expected: "refunded"},
		{name: "partial refund", status: "partial_refund", expected: "refunded"},
		{name: "chargeback", status: "chargeback", expected: "refunded"},
		{name: "partial chargeback", status: "partial_chargeback", expected: "refunded"},
		{name: "unknown", status: "authorize", expected: "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWebhookStatus(WebhookNotification{TransactionStatus: tt.status, FraudStatus: tt.fraud})
			if got != tt.expected {
				t.Fatalf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
