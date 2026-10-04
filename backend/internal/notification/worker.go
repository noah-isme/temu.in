package notification

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"regexp"
	"time"

	"temu.in/backend/internal/models"
)

type WorkerConfig struct {
	PollInterval     time.Duration
	ReminderInterval time.Duration
	BatchSize        int
	MaxAttempts      int
	RetryBase        time.Duration
}

type OutboxRepository interface {
	ClaimBatch(ctx context.Context, now time.Time, limit int) ([]models.Notification, error)
	MarkSent(ctx context.Context, notification models.Notification, providerMessageID string) error
	// MarkFailure reports deadLettered=true only when this call moved the row
	// to dead_letter.
	MarkFailure(ctx context.Context, notification models.Notification, deliveryErr error, nextAttempt time.Time, maxAttempts int) (deadLettered bool, err error)
}

type Worker struct {
	// Logger is optional; nil uses slog.Default().
	Logger *slog.Logger

	repo             OutboxRepository
	sender           EmailSender
	pollInterval     time.Duration
	reminderInterval time.Duration
	batchSize        int
	maxAttempts      int
	retryBase        time.Duration
	reminderService  ReminderScheduler
	lastReminderRun  time.Time
}

func NewWorker(repo OutboxRepository, sender EmailSender, cfg WorkerConfig, reminderServices ...ReminderScheduler) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 25
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 8
	}
	if cfg.RetryBase <= 0 {
		cfg.RetryBase = 15 * time.Second
	}
	if cfg.ReminderInterval <= 0 {
		cfg.ReminderInterval = time.Hour
	}
	var reminderService ReminderScheduler
	if len(reminderServices) > 0 {
		reminderService = reminderServices[0]
	}
	return &Worker{
		repo:             repo,
		sender:           sender,
		pollInterval:     cfg.PollInterval,
		reminderInterval: cfg.ReminderInterval,
		batchSize:        cfg.BatchSize,
		maxAttempts:      cfg.MaxAttempts,
		retryBase:        cfg.RetryBase,
		reminderService:  reminderService,
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) error {
	if w == nil || w.repo == nil {
		return errors.New("notification worker is unavailable")
	}
	now := time.Now().UTC()
	if w.reminderService != nil && (w.lastReminderRun.IsZero() || now.Sub(w.lastReminderRun) >= w.reminderInterval) {
		if err := w.reminderService.EnqueueUpcomingReminders(ctx, now); err != nil {
			return err
		}
		w.lastReminderRun = now
	}
	claimed, err := w.repo.ClaimBatch(ctx, now, w.batchSize)
	if err != nil {
		return err
	}
	for _, notification := range claimed {
		if err := w.deliver(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) deliver(ctx context.Context, notification models.Notification) error {
	message, err := RenderNotification(notification)
	if err == nil {
		if w.sender == nil {
			err = ErrProviderNotConfigured
		} else {
			var providerMessageID string
			providerMessageID, err = w.sender.Send(ctx, message)
			if err == nil {
				return w.repo.MarkSent(ctx, notification, providerMessageID)
			}
		}
	}

	if err == nil {
		err = errors.New("notification delivery failed")
	}
	// Provider errors can echo the recipient address; never log or persist it.
	redacted := errors.New(redactEmails(err.Error()))
	nextAttempt := time.Now().UTC().Add(w.retryDelay(notification.Attempts))
	deadLettered, markErr := w.repo.MarkFailure(ctx, notification, redacted, nextAttempt, w.maxAttempts)
	if markErr != nil {
		return markErr
	}
	attrs := []any{
		"notification_id", notification.ID,
		"type", notification.Type,
		"template", notification.Template,
		"attempts", notification.Attempts,
		"error", redacted.Error(),
	}
	if notification.BookingID != nil {
		attrs = append(attrs, "booking_id", *notification.BookingID)
	}
	if deadLettered {
		w.logger().ErrorContext(ctx, "notification dead-lettered", attrs...)
		return nil
	}
	if notification.Attempts < w.maxAttempts {
		w.logger().WarnContext(ctx, "notification delivery failed, scheduled retry", append(attrs, "next_attempt_at", nextAttempt)...)
	}
	return nil
}

func (w *Worker) logger() *slog.Logger {
	if w.Logger != nil {
		return w.Logger
	}
	return slog.Default()
}

var emailPattern = regexp.MustCompile(`[^\s@"]+@[^\s@"]+`)

// redactEmails replaces anything shaped like an email address with [redacted].
func redactEmails(s string) string {
	return emailPattern.ReplaceAllString(s, "[redacted]")
}

func (w *Worker) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	seconds := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(w.retryBase) * seconds)
	maxDelay := 24 * time.Hour
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// Run is the managed process-level worker. The router owns its lifecycle via
// context cancellation, so delivery is not tied to an HTTP request lifetime.
func (w *Worker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("notification worker is unavailable")
	}
	if err := w.ProcessOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		w.logger().ErrorContext(ctx, "notification worker cycle failed", "error", redactEmails(err.Error()))
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.ProcessOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				w.logger().ErrorContext(ctx, "notification worker cycle failed", "error", redactEmails(err.Error()))
			}
		}
	}
}
