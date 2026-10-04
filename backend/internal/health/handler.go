package health

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"

	"temu.in/backend/internal/logging"
)

// DefaultTimeout bounds the whole readiness probe.
const DefaultTimeout = 2 * time.Second

const (
	StatusOK          = "ok"
	StatusDegraded    = "degraded"
	StatusUnavailable = "unavailable"
)

// Pinger checks one dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingFunc adapts a function to Pinger.
type PingFunc func(ctx context.Context) error

func (f PingFunc) Ping(ctx context.Context) error { return f(ctx) }

type namedCheck struct {
	name     string
	pinger   Pinger
	critical bool
}

// Handler serves liveness and readiness probes.
type Handler struct {
	checks  []namedCheck
	timeout time.Duration
	logger  *slog.Logger
}

// CheckResult is one dependency entry in the readiness body. It never carries
// error text (driver errors can leak hostnames or credentials).
type CheckResult struct {
	Status    string `json:"status"`
	LatencyMS *int64 `json:"latency_ms,omitempty"`
	Critical  bool   `json:"critical"`
}

// ReadinessResponse is the readiness body.
type ReadinessResponse struct {
	Status    string                 `json:"status"`
	CheckedAt time.Time              `json:"checked_at"`
	Checks    map[string]CheckResult `json:"checks"`
}

// NewHandler returns a handler with no checks; timeout <= 0 uses DefaultTimeout.
func NewHandler(timeout time.Duration) *Handler {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Handler{timeout: timeout}
}

// With registers a dependency check. A nil pinger (including a typed nil) is skipped.
func (h *Handler) With(name string, p Pinger, critical bool) *Handler {
	if isNil(p) {
		return h
	}
	h.checks = append(h.checks, namedCheck{name: name, pinger: p, critical: critical})
	return h
}

// WithLogger overrides the logger used for failure details (default slog.Default()).
func (h *Handler) WithLogger(l *slog.Logger) *Handler {
	h.logger = l
	return h
}

func isNil(p Pinger) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func (h *Handler) log() *slog.Logger {
	if h.logger != nil {
		return h.logger
	}
	return slog.Default()
}

type pingResult struct {
	err     error
	latency time.Duration
}

// Readiness pings every dependency concurrently under one deadline. A failing
// critical dependency yields 503 "unavailable"; only non-critical failures
// yield 200 "degraded".
func (h *Handler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	results := make([]chan pingResult, len(h.checks))
	for i, chk := range h.checks {
		ch := make(chan pingResult, 1) // buffered: an abandoned pinger never blocks
		results[i] = ch
		go func(p Pinger) {
			start := time.Now()
			err := p.Ping(ctx)
			ch <- pingResult{err: err, latency: time.Since(start)}
		}(chk.pinger)
	}

	body := ReadinessResponse{Status: StatusOK, CheckedAt: time.Now().UTC(), Checks: map[string]CheckResult{}}
	code := http.StatusOK
	requestID := logging.RequestIDFromContext(c.Request.Context())
	for i, chk := range h.checks {
		var res pingResult
		select {
		case res = <-results[i]:
		case <-ctx.Done():
			res = pingResult{err: ctx.Err()}
		}

		entry := CheckResult{Status: StatusOK, Critical: chk.critical}
		if res.err != nil {
			entry.Status = StatusUnavailable
			h.log().Warn("readiness check failed",
				"dependency", chk.name, "critical", chk.critical, "error", res.err.Error(), "request_id", requestID)
			if chk.critical {
				body.Status = StatusUnavailable
				code = http.StatusServiceUnavailable
			} else if body.Status == StatusOK {
				body.Status = StatusDegraded
			}
		} else {
			ms := res.latency.Milliseconds()
			entry.LatencyMS = &ms
		}
		body.Checks[chk.name] = entry
	}

	c.JSON(code, body)
}

// Liveness reports that the process is up.
func (h *Handler) Liveness(c *gin.Context) {
	liveness(c)
}

func liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}
