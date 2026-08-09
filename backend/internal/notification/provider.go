package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrProviderNotConfigured = errors.New("sendgrid provider is not configured")

// EmailMessage is the rendered representation sent by a delivery worker.
// TemplateName is retained for observability and for selecting an optional
// SendGrid dynamic template ID.
type EmailMessage struct {
	To           string
	ToName       string
	Subject      string
	Text         string
	HTML         string
	TemplateName string
	TemplateID   string
	TemplateData map[string]string
}

type EmailSender interface {
	Send(ctx context.Context, message EmailMessage) (string, error)
}

type SendGridProvider struct {
	APIKey      string
	FromEmail   string
	Endpoint    string
	HTTPClient  *http.Client
	TemplateIDs map[string]string
}

func NewSendGridProvider(apiKey, fromEmail string, templateIDs map[string]string) *SendGridProvider {
	return &SendGridProvider{
		APIKey:      strings.TrimSpace(apiKey),
		FromEmail:   strings.TrimSpace(fromEmail),
		Endpoint:    "https://api.sendgrid.com/v3/mail/send",
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
		TemplateIDs: templateIDs,
	}
}

func (p *SendGridProvider) Send(ctx context.Context, message EmailMessage) (string, error) {
	if p == nil || strings.TrimSpace(p.APIKey) == "" || strings.TrimSpace(p.FromEmail) == "" {
		return "", ErrProviderNotConfigured
	}
	if strings.TrimSpace(message.To) == "" {
		return "", errors.New("recipient email is required")
	}
	if strings.TrimSpace(message.TemplateID) == "" && p.TemplateIDs != nil {
		message.TemplateID = p.TemplateIDs[message.TemplateName]
	}

	personalization := map[string]interface{}{
		"to": []map[string]string{{"email": message.To, "name": message.ToName}},
	}
	if len(message.TemplateData) > 0 {
		personalization["dynamic_template_data"] = message.TemplateData
	}

	body := map[string]interface{}{
		"personalizations": []map[string]interface{}{personalization},
		"from":             map[string]string{"email": p.FromEmail},
		"subject":          message.Subject,
	}
	if message.TemplateID != "" {
		body["template_id"] = message.TemplateID
	} else {
		body["content"] = []map[string]string{
			{"type": "text/plain", "value": message.Text},
			{"type": "text/html", "value": message.HTML},
		}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal SendGrid request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create SendGrid request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+p.APIKey)
	request.Header.Set("Content-Type", "application/json")

	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send SendGrid request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		messageBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("SendGrid returned %s: %s", response.Status, strings.TrimSpace(string(messageBody)))
	}

	return response.Header.Get("X-Message-Id"), nil
}
