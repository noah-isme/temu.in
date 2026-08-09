package notification

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"

	"temu.in/backend/internal/models"
)

// RenderNotification turns the stored logical template and JSON payload into
// a provider-neutral message. Templates stay versioned in code while the
// outbox keeps the exact data used for a delivery attempt.
func RenderNotification(notification models.Notification) (EmailMessage, error) {
	data := make(map[string]string)
	if notification.Payload != "" {
		if err := json.Unmarshal([]byte(notification.Payload), &data); err != nil {
			return EmailMessage{}, fmt.Errorf("decode notification payload: %w", err)
		}
	}
	if notification.RecipientName != "" {
		data["recipient_name"] = notification.RecipientName
	}

	name := data["recipient_name"]
	if name == "" {
		name = "there"
	}

	text := ""
	switch notification.Template {
	case models.NotificationTypeBookingConfirmationCustomer:
		text = fmt.Sprintf("Hi %s,\n\nYour booking with %s is confirmed for %s at %s-%s.\n\nThank you for choosing Temu.in!", name, data["business_name"], data["booking_date"], data["start_time"], data["end_time"])
	case models.NotificationTypeBookingConfirmationProvider:
		text = fmt.Sprintf("A booking for %s on %s at %s-%s has been confirmed.", data["customer_name"], data["booking_date"], data["start_time"], data["end_time"])
	case models.NotificationTypeBookingCancellationCustomer:
		text = fmt.Sprintf("Hi %s,\n\nYour booking with %s on %s has been cancelled.\nReason: %s", name, data["business_name"], data["booking_date"], data["reason"])
	case models.NotificationTypeBookingCancellationProvider:
		text = fmt.Sprintf("The booking for %s on %s at %s has been cancelled.\nReason: %s", data["customer_name"], data["booking_date"], data["start_time"], data["reason"])
	case models.NotificationTypeBookingReminderCustomer:
		text = fmt.Sprintf("Hi %s,\n\nReminder: you have a booking on %s at %s-%s.", name, data["booking_date"], data["start_time"], data["end_time"])
	case models.NotificationTypeBookingReminderProvider:
		text = fmt.Sprintf("Reminder: %s has a booking on %s at %s-%s.", data["customer_name"], data["booking_date"], data["start_time"], data["end_time"])
	case models.NotificationTypePaymentReceipt:
		text = fmt.Sprintf("Hi %s,\n\nWe received your payment of %s %s for booking %s.\nOrder: %s\nTransaction: %s", name, data["currency"], data["amount"], data["booking_id"], data["order_id"], data["transaction_id"])
	case models.NotificationTypePasswordReset:
		text = fmt.Sprintf("Hi %s,\n\nReset your password here:\n%s\n\nIf you did not request this, you can ignore this email.", name, data["reset_link"])
	case models.NotificationTypeEmailVerification:
		text = fmt.Sprintf("Hi %s,\n\nVerify your Temu.in email here:\n%s", name, data["verify_link"])
	default:
		return EmailMessage{}, errors.New("unknown notification template: " + notification.Template)
	}

	var htmlBody string
	paragraphs := strings.Split(text, "\n\n")
	for _, paragraph := range paragraphs {
		htmlBody += "<p>" + strings.ReplaceAll(html.EscapeString(paragraph), "\n", "<br>") + "</p>"
	}
	return EmailMessage{
		To:           notification.RecipientEmail,
		ToName:       notification.RecipientName,
		Subject:      notification.Subject,
		Text:         text,
		HTML:         "<html><body>" + htmlBody + "</body></html>",
		TemplateName: notification.Template,
		TemplateData: data,
	}, nil
}
