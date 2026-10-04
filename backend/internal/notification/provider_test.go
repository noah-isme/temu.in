package notification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendGridProviderSendsDynamicTemplate(t *testing.T) {
	var requestBody map[string]interface{}

	provider := NewSendGridProvider("test-key", "no-reply@example.test", map[string]string{"booking_confirmation_customer": "d-template"})
	provider.Endpoint = "https://sendgrid.test/v3/mail/send"
	provider.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Status:     "202 Accepted",
			Header:     http.Header{"X-Message-Id": []string{"message-1"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})}
	messageID, err := provider.Send(context.Background(), EmailMessage{
		To:           "customer@example.test",
		Subject:      "Booking confirmed",
		TemplateName: "booking_confirmation_customer",
		TemplateData: map[string]string{"booking_id": "12"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if messageID != "message-1" {
		t.Fatalf("message id = %q", messageID)
	}
	if requestBody["template_id"] != "d-template" {
		t.Fatalf("template id = %#v", requestBody["template_id"])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSendGridProviderRequiresConfiguration(t *testing.T) {
	provider := NewSendGridProvider("", "no-reply@example.test", nil)
	if _, err := provider.Send(context.Background(), EmailMessage{To: "customer@example.test"}); err != ErrProviderNotConfigured {
		t.Fatalf("error = %v, want ErrProviderNotConfigured", err)
	}
}

// sendAndCapture posts one message through provider to an httptest server and
// returns the decoded request body SendGrid would have received.
func sendAndCapture(t *testing.T, sandbox bool) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewSendGridProvider("test-key", "no-reply@example.test", nil)
	provider.Endpoint = server.URL + "/v3/mail/send"
	provider.HTTPClient = server.Client()
	provider.SandboxMode = sandbox
	if _, err := provider.Send(context.Background(), EmailMessage{To: "customer@example.test", Subject: "s", Text: "t", HTML: "<p>t</p>"}); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestSendGridSandboxMode(t *testing.T) {
	body := sendAndCapture(t, true)
	settings, ok := body["mail_settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("mail_settings missing: %v", body)
	}
	sandbox, ok := settings["sandbox_mode"].(map[string]interface{})
	if !ok || sandbox["enable"] != true {
		t.Fatalf("mail_settings.sandbox_mode.enable = %v, want true", settings["sandbox_mode"])
	}

	if body := sendAndCapture(t, false); body["mail_settings"] != nil {
		t.Fatalf("mail_settings present without sandbox mode: %v", body["mail_settings"])
	}
}
