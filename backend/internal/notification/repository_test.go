package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

func TestIntegrationMarkFailureDeadLetterTransition(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	n := &models.Notification{
		Type:             models.NotificationTypeEmailVerification,
		DeduplicationKey: "mark-failure-test",
		RecipientEmail:   "r@example.test",
		Template:         models.NotificationTypeEmailVerification,
	}
	if err := repo.Enqueue(ctx, n); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := db.Model(&models.Notification{}).Where("id = ?", n.ID).
		Updates(map[string]any{"status": models.NotificationStatusProcessing, "attempts": 3}).Error; err != nil {
		t.Fatalf("set processing: %v", err)
	}
	n.Attempts = 3

	dead, err := repo.MarkFailure(ctx, *n, errors.New("boom"), time.Now().Add(time.Minute), 3)
	if err != nil || !dead {
		t.Fatalf("first MarkFailure = (%v, %v), want (true, nil)", dead, err)
	}
	var stored models.Notification
	if err := db.First(&stored, n.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != models.NotificationStatusDeadLetter || stored.LastError != "boom" {
		t.Fatalf("stored status=%q last_error=%q", stored.Status, stored.LastError)
	}

	dead, err = repo.MarkFailure(ctx, *n, errors.New("boom again"), time.Now().Add(time.Minute), 3)
	if err != nil || dead {
		t.Fatalf("second MarkFailure = (%v, %v), want (false, nil)", dead, err)
	}
}

func TestIntegrationMarkFailureRetryIsNotDeadLetter(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	n := &models.Notification{
		Type:             models.NotificationTypeEmailVerification,
		DeduplicationKey: "mark-failure-retry",
		RecipientEmail:   "r@example.test",
		Template:         models.NotificationTypeEmailVerification,
	}
	if err := repo.Enqueue(ctx, n); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := db.Model(&models.Notification{}).Where("id = ?", n.ID).
		Updates(map[string]any{"status": models.NotificationStatusProcessing, "attempts": 1}).Error; err != nil {
		t.Fatalf("set processing: %v", err)
	}
	n.Attempts = 1
	dead, err := repo.MarkFailure(ctx, *n, errors.New("temporary"), time.Now().Add(time.Minute), 3)
	if err != nil || dead {
		t.Fatalf("MarkFailure = (%v, %v), want (false, nil)", dead, err)
	}
	var stored models.Notification
	if err := db.First(&stored, n.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != models.NotificationStatusRetry {
		t.Fatalf("status = %q, want retry", stored.Status)
	}
}
