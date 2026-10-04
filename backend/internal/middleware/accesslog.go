package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DefaultAccessLogSkipPrefixes keeps health probes out of the access log.
var DefaultAccessLogSkipPrefixes = HealthPathPrefixes

// AccessLog writes one structured line per request after it completes.
// skipPrefixes defaults to DefaultAccessLogSkipPrefixes when none are given.
func AccessLog(logger *slog.Logger, skipPrefixes ...string) gin.HandlerFunc {
	if len(skipPrefixes) == 0 {
		skipPrefixes = DefaultAccessLogSkipPrefixes
	}
	return func(c *gin.Context) {
		start := time.Now()
		rawPath := c.Request.URL.Path
		c.Next()

		if hasAnyPrefix(rawPath, skipPrefixes) {
			return
		}
		l := logger
		if l == nil {
			l = slog.Default()
		}
		path := c.FullPath()
		if path == "" {
			path = rawPath
		}
		status := c.Writer.Status()
		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		// request_id is attached explicitly so the line carries it with any handler.
		l.LogAttrs(context.Background(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("request_id", RequestIDFromGin(c)),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Int("bytes", size),
		)
	}
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
