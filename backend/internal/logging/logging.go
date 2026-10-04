// Package logging builds the process-wide structured logger (log/slog).
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// IsDevEnv reports whether appEnv is a local/development environment, which
// gets human-readable text logs instead of JSON.
func IsDevEnv(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "development", "dev", "local", "test":
		return true
	default:
		return false
	}
}

// ParseLevel maps debug|info|warn|error (case-insensitive) to a slog.Level.
// Anything else yields slog.LevelInfo.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New returns a logger writing to stdout: text in development environments,
// JSON otherwise.
func New(appEnv, level string) *slog.Logger {
	return NewWithWriter(os.Stdout, appEnv, level)
}

// NewWithWriter is New with an explicit destination (used by tests).
func NewWithWriter(w io.Writer, appEnv, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: ParseLevel(level), AddSource: false}
	var base slog.Handler
	if IsDevEnv(appEnv) {
		base = slog.NewTextHandler(w, opts)
	} else {
		base = slog.NewJSONHandler(w, opts)
	}
	return slog.New(&contextHandler{inner: base})
}
