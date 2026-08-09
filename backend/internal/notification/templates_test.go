package notification

import (
	"strings"
	"testing"

	"temu.in/backend/internal/models"
)

func TestRenderNotificationUsesStoredPayloadAndEscapesHTML(t *testing.T) {
	message, err := RenderNotification(models.Notification{
		RecipientEmail: "customer@example.test",
		RecipientName:  "Ada <script>",
		Subject:        "Booking confirmed",
		Template:       models.NotificationTypeBookingConfirmationCustomer,
		Payload:        `{"business_name":"Clinic & Co","booking_date":"2026-08-10","start_time":"09:00","end_time":"10:00"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.To != "customer@example.test" || message.Subject != "Booking confirmed" {
		t.Fatalf("unexpected message metadata: %#v", message)
	}
	if !strings.Contains(message.Text, "Ada <script>") {
		t.Fatalf("plain text should preserve the recipient name: %q", message.Text)
	}
	if strings.Contains(message.HTML, "<script>") || !strings.Contains(message.HTML, "&lt;script&gt;") {
		t.Fatalf("html body was not escaped: %q", message.HTML)
	}
}

func TestRenderNotificationRejectsUnknownTemplate(t *testing.T) {
	if _, err := RenderNotification(models.Notification{Template: "missing"}); err == nil {
		t.Fatal("expected unknown template error")
	}
}
