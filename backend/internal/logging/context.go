package logging

import (
	"context"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID returns a context carrying the request ID, which loggers built
// by New attach to every record logged with that context.
func WithRequestID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext returns the request ID stored by WithRequestID, or "".
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// contextHandler adds request_id from the record's context before delegating,
// so slog.InfoContext(ctx, ...) calls carry it automatically.
type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, rec slog.Record) error {
	if id := RequestIDFromContext(ctx); id != "" {
		rec = rec.Clone()
		rec.AddAttrs(slog.String("request_id", id))
	}
	return h.inner.Handle(ctx, rec)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}
