package notification

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestSendGridSandbox sends one real request to SendGrid with sandbox mode on,
// so SendGrid validates it and answers 200 without delivering mail. It is
// opt-in: it skips unless SENDGRID_SANDBOX_API_KEY, SENDGRID_SANDBOX_FROM_EMAIL
// (a verified sender) and SENDGRID_SANDBOX_TO_EMAIL are all set.
func TestSendGridSandbox(t *testing.T) {
	apiKey := os.Getenv("SENDGRID_SANDBOX_API_KEY")
	if apiKey == "" {
		t.Skip("SENDGRID_SANDBOX_API_KEY not set; skipping SendGrid sandbox test")
	}
	from, to := os.Getenv("SENDGRID_SANDBOX_FROM_EMAIL"), os.Getenv("SENDGRID_SANDBOX_TO_EMAIL")
	if from == "" || to == "" {
		t.Skip("SENDGRID_SANDBOX_FROM_EMAIL and SENDGRID_SANDBOX_TO_EMAIL are required for the SendGrid sandbox test")
	}

	provider := NewSendGridProvider(apiKey, from, nil)
	provider.SandboxMode = true
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	message := EmailMessage{
		To:      to,
		ToName:  "Sandbox Recipient",
		Subject: "temu.in SendGrid sandbox test",
		Text:    "Sandbox mode: this message is validated but never delivered.",
		HTML:    "<p>Sandbox mode: this message is validated but never delivered.</p>",
	}
	if _, err := provider.Send(ctx, message); err != nil {
		t.Fatalf("sandbox send: %v", err)
	}
}
