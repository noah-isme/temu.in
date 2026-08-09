package notification

import (
	"context"
	"errors"
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
}

func (f *fakeOutboxRepository) ClaimBatch(context.Context, time.Time, int) ([]models.Notification, error) {
	return nil, nil
}

func (f *fakeOutboxRepository) MarkSent(_ context.Context, _ models.Notification, providerMessageID string) error {
	f.markedSent = true
	f.providerID = providerMessageID
	return nil
}

func (f *fakeOutboxRepository) MarkFailure(_ context.Context, _ models.Notification, deliveryErr error, _ time.Time, maxAttempts int) error {
	f.markedFailure = true
	f.failure = deliveryErr
	f.maxAttempts = maxAttempts
	return nil
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
