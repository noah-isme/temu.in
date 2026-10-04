package payment

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"testing"
	"time"
)

// TestMidtransSandbox talks to the real Midtrans sandbox. It is opt-in: it
// skips unless MIDTRANS_SANDBOX_SERVER_KEY is set (never in CI by default).
// MIDTRANS_SANDBOX_CLIENT_KEY is optional.
func TestMidtransSandbox(t *testing.T) {
	serverKey := os.Getenv("MIDTRANS_SANDBOX_SERVER_KEY")
	if serverKey == "" {
		t.Skip("MIDTRANS_SANDBOX_SERVER_KEY not set; skipping Midtrans sandbox test")
	}
	provider := NewMidtransProvider(serverKey, os.Getenv("MIDTRANS_SANDBOX_CLIENT_KEY"), "https://api.sandbox.midtrans.com")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("create transaction", func(t *testing.T) {
		orderID, err := NewOrderID()
		if err != nil {
			t.Fatal(err)
		}
		tx, err := provider.CreateTransaction(ctx, CreateTransactionRequest{OrderID: orderID, Amount: 10000, Currency: "IDR", Description: "temu.in sandbox test"})
		if err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
		if tx.Token == "" || tx.RedirectURL == "" {
			t.Fatalf("sandbox returned token=%q redirect_url=%q", tx.Token, tx.RedirectURL)
		}
	})

	t.Run("status of an unpaid order", func(t *testing.T) {
		orderID, err := NewOrderID()
		if err != nil {
			t.Fatal(err)
		}
		// Midtrans answers 404 until a payment is attempted; only the error
		// is asserted, not a status.
		if _, err := provider.GetTransaction(ctx, orderID); err == nil {
			t.Fatal("expected an error for a never-paid order")
		}
	})

	t.Run("verify webhook signature", func(t *testing.T) {
		n := WebhookNotification{OrderID: "temu_sandbox", StatusCode: "200", GrossAmount: "10000.00", TransactionStatus: "settlement"}
		digest := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + serverKey))
		n.SignatureKey = hex.EncodeToString(digest[:])
		if !provider.VerifyWebhook(n) {
			t.Fatal("locally signed notification rejected")
		}
	})
}
