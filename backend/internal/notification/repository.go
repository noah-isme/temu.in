package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"temu.in/backend/internal/models"
)

const defaultLockTimeout = 10 * time.Minute

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Enqueue is idempotent by business event. This makes webhook replays and
// client retries safe without sending the same email more than once.
func (r *Repository) Enqueue(ctx context.Context, notification *models.Notification) error {
	if r == nil || r.db == nil {
		return errors.New("notification repository is unavailable")
	}
	if notification == nil {
		return errors.New("notification is required")
	}
	if strings.TrimSpace(notification.DeduplicationKey) == "" {
		return errors.New("notification deduplication key is required")
	}
	if notification.Channel == "" {
		notification.Channel = models.NotificationChannelEmail
	}
	if notification.Status == "" {
		notification.Status = models.NotificationStatusPending
	}
	if notification.Payload == "" {
		notification.Payload = "{}"
	}
	if notification.NextAttemptAt == nil {
		now := time.Now().UTC()
		notification.NextAttemptAt = &now
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "deduplication_key"}},
		DoNothing: true,
	}).Create(notification)
	return result.Error
}

// ClaimBatch atomically claims pending work. FOR UPDATE SKIP LOCKED allows
// multiple application instances to run workers without duplicate delivery.
func (r *Repository) ClaimBatch(ctx context.Context, now time.Time, limit int) ([]models.Notification, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("notification repository is unavailable")
	}
	if limit <= 0 {
		limit = 25
	}

	staleBefore := now.Add(-defaultLockTimeout)
	claimed := make([]models.Notification, 0, limit)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []models.Notification
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(`
				(channel = ? AND (
					(status IN ? AND next_attempt_at <= ?) OR
					(status = ? AND locked_at < ?)
				))`, models.NotificationChannelEmail,
				[]string{models.NotificationStatusPending, models.NotificationStatusRetry}, now,
				models.NotificationStatusProcessing, staleBefore).
			Order("id ASC").Limit(limit).Find(&candidates).Error; err != nil {
			return err
		}

		for i := range candidates {
			notification := &candidates[i]
			lockedAt := now
			result := tx.Model(&models.Notification{}).
				Where("id = ?", notification.ID).
				Updates(map[string]interface{}{
					"status":     models.NotificationStatusProcessing,
					"locked_at":  lockedAt,
					"attempts":   gorm.Expr("attempts + 1"),
					"updated_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				continue
			}
			notification.Status = models.NotificationStatusProcessing
			notification.LockedAt = &lockedAt
			notification.Attempts++
			claimed = append(claimed, *notification)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("claim notifications: %w", err)
	}
	return claimed, nil
}

func (r *Repository) MarkSent(ctx context.Context, notification models.Notification, providerMessageID string) error {
	if r == nil || r.db == nil {
		return errors.New("notification repository is unavailable")
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Notification{}).Where("id = ? AND status = ?", notification.ID, models.NotificationStatusProcessing).Updates(map[string]interface{}{
			"status":              models.NotificationStatusSent,
			"locked_at":           nil,
			"sent_at":             now,
			"provider_message_id": providerMessageID,
			"last_error":          "",
			"updated_at":          now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		return tx.Create(&models.NotificationDelivery{
			NotificationID:    notification.ID,
			Attempt:           notification.Attempts,
			Status:            models.NotificationDeliverySent,
			ProviderMessageID: providerMessageID,
		}).Error
	})
}

// MarkFailure records a failed delivery attempt. deadLettered is true only when
// this call moved the row from processing to dead_letter; a worker that lost
// the processing race (row no longer in processing) gets false.
func (r *Repository) MarkFailure(ctx context.Context, notification models.Notification, deliveryErr error, nextAttempt time.Time, maxAttempts int) (deadLettered bool, err error) {
	if r == nil || r.db == nil {
		return false, errors.New("notification repository is unavailable")
	}
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	if deliveryErr == nil {
		deliveryErr = errors.New("notification delivery failed")
	}
	status := models.NotificationStatusRetry
	deliveryStatus := models.NotificationDeliveryRetry
	if notification.Attempts >= maxAttempts {
		status = models.NotificationStatusDeadLetter
		deliveryStatus = models.NotificationDeliveryDeadLetter
	}
	errorText := deliveryErr.Error()
	if len(errorText) > 4000 {
		errorText = errorText[:4000]
	}
	now := time.Now().UTC()
	transitioned := false
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Notification{}).Where("id = ? AND status = ?", notification.ID, models.NotificationStatusProcessing).Updates(map[string]interface{}{
			"status":          status,
			"locked_at":       nil,
			"next_attempt_at": nextAttempt,
			"last_error":      errorText,
			"updated_at":      now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		transitioned = true
		return tx.Create(&models.NotificationDelivery{
			NotificationID: notification.ID,
			Attempt:        notification.Attempts,
			Status:         deliveryStatus,
			Error:          errorText,
		}).Error
	})
	if err != nil {
		return false, err
	}
	return transitioned && status == models.NotificationStatusDeadLetter, nil
}
