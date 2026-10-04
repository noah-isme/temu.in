package notification

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"temu.in/backend/internal/models"
)

type fakeOutboxRepository struct {
	markedSent    bool
	providerID    string
	markedFailure bool
	failure       error
	maxAttempts   int
	deadLettered  bool
}

func (f *fakeOutboxRepository) ClaimBatch(context.Context, time.Time, int) ([]models.Notification, error) {
	return nil, nil
}

func (f *fakeOutboxRepository) MarkSent(_ context.Context, _ models.Notification, providerMessageID string) error {
	f.markedSent = true
	f.providerID = providerMessageID
	return nil
}

func (f *fakeOutboxRepository) MarkFailure(_ context.Context, _ models.Notification, deliveryErr error, _ time.Time, maxAttempts int) (bool, error) {
	f.markedFailure = true
	f.failure = deliveryErr
	f.maxAttempts = maxAttempts
	return f.deadLettered, nil
}

type failingEmailSender struct{}

func (failingEmailSender) Send(context.Context, EmailMessage) (string, error) {
	return "", errors.New("temporary provider outage")
}

type successfulEmailSender struct{}

func (successfulEmailSender) Send(context.Context, EmailMessage) (string, error) {
	return "provider-message-1", nil
}

func TestWorkerDeadLettersAfterMaximumAttempts(t *testing.T) {
	repo := &fakeOutboxRepository{}
	worker := NewWorker(repo, failingEmailSender{}, WorkerConfig{MaxAttempts: 3, RetryBase: time.Second})
	notification := models.Notification{
		ID:             1,
		Attempts:       3,
		RecipientEmail: "customer@example.test",
		Template:       models.NotificationTypeEmailVerification,
		Payload:        `{"verify_link":"https://example.test/verify"}`,
	}
	if err := worker.deliver(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	if !repo.markedFailure || repo.maxAttempts != 3 {
		t.Fatalf("failure was not recorded with max attempts: %#v", repo)
	}
	if repo.failure == nil || repo.failure.Error() != "temporary provider outage" {
		t.Fatalf("unexpected failure: %v", repo.failure)
	}
}

func TestWorkerMarksSuccessfulDelivery(t *testing.T) {
	repo := &fakeOutboxRepository{}
	worker := NewWorker(repo, successfulEmailSender{}, WorkerConfig{})
	notification := models.Notification{
		ID:             2,
		Attempts:       1,
		RecipientEmail: "customer@example.test",
		Template:       models.NotificationTypeEmailVerification,
		Payload:        `{"verify_link":"https://example.test/verify"}`,
	}
	if err := worker.deliver(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	if !repo.markedSent || repo.providerID != "provider-message-1" {
		t.Fatalf("success was not recorded: %#v", repo)
	}
}

type leakyEmailSender struct{}

func (leakyEmailSender) Send(context.Context, EmailMessage) (string, error) {
	return "", errors.New(`SendGrid returned 400 Bad Request: {"errors":[{"message":"invalid to: someone@example.com"}]}`)
}

func testNotification(attempts int) models.Notification {
	bookingID := uint(42)
	return models.Notification{
		ID:             9,
		Type:           models.NotificationTypeEmailVerification,
		Attempts:       attempts,
		RecipientEmail: "customer@example.test",
		Template:       models.NotificationTypeEmailVerification,
		Payload:        `{"verify_link":"https://example.test/verify"}`,
		BookingID:      &bookingID,
	}
}

func newLoggedWorker(repo OutboxRepository, sender EmailSender, buf *bytes.Buffer) *Worker {
	w := NewWorker(repo, sender, WorkerConfig{MaxAttempts: 3, RetryBase: time.Second})
	w.Logger = slog.New(slog.NewTextHandler(buf, nil))
	return w
}

func TestWorkerLogsErrorWhenDeadLettered(t *testing.T) {
	var buf bytes.Buffer
	repo := &fakeOutboxRepository{deadLettered: true}
	if err := newLoggedWorker(repo, failingEmailSender{}, &buf).deliver(context.Background(), testNotification(3)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, `msg="notification dead-lettered"`) {
		t.Fatalf("expected dead-letter ERROR log, got %q", out)
	}
	for _, want := range []string{"notification_id=9", "attempts=3", "booking_id=42", "template=", "type="} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "customer@example.test") {
		t.Fatalf("log leaks recipient: %q", out)
	}
}

func TestWorkerFirstFailureLogsWarnOnly(t *testing.T) {
	var buf bytes.Buffer
	repo := &fakeOutboxRepository{}
	if err := newLoggedWorker(repo, failingEmailSender{}, &buf).deliver(context.Background(), testNotification(1)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "level=ERROR") {
		t.Fatalf("first failure must not log ERROR: %q", out)
	}
	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, "scheduled retry") || !strings.Contains(out, "next_attempt_at=") {
		t.Fatalf("expected retry WARN, got %q", out)
	}
}

func TestWorkerLostRaceLogsNoError(t *testing.T) {
	// The repo returns (false, nil): the row was no longer processing, so this
	// worker did not dead-letter it even though attempts reached the maximum.
	var buf bytes.Buffer
	repo := &fakeOutboxRepository{deadLettered: false}
	if err := newLoggedWorker(repo, failingEmailSender{}, &buf).deliver(context.Background(), testNotification(3)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("lost race must not log ERROR: %q", buf.String())
	}
}

func TestWorkerRedactsEmailsInLogsAndPersistedError(t *testing.T) {
	var buf bytes.Buffer
	repo := &fakeOutboxRepository{deadLettered: true}
	if err := newLoggedWorker(repo, leakyEmailSender{}, &buf).deliver(context.Background(), testNotification(3)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "someone@example.com") || !strings.Contains(buf.String(), "[redacted]") {
		t.Fatalf("log not redacted: %q", buf.String())
	}
	if repo.failure == nil || strings.Contains(repo.failure.Error(), "someone@example.com") || !strings.Contains(repo.failure.Error(), "[redacted]") {
		t.Fatalf("persisted error not redacted: %v", repo.failure)
	}
}

func TestRedactEmails(t *testing.T) {
	cases := map[string]string{
		"plain error":                    "plain error",
		"bad to: someone@example.com":    "bad to: [redacted]",
		`{"email":"a.b+c@d.co"} and x@y`: `{"email":"[redacted]"} and [redacted]`,
	}
	for in, want := range cases {
		if got := redactEmails(in); got != want {
			t.Errorf("redactEmails(%q) = %q, want %q", in, got, want)
		}
	}
}
