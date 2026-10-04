package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

type fakePinger struct{ err error }

func (f *fakePinger) Ping(context.Context) error { return f.err }

func serveReady(t *testing.T, h *Handler) (*httptest.ResponseRecorder, ReadinessResponse) {
	t.Helper()
	r := gin.New()
	RegisterRoutes(r.Group("/health"), h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	var body ReadinessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return w, body
}

func TestReadinessAllOK(t *testing.T) {
	h := NewHandler(time.Second).
		With("postgres", &fakePinger{}, true).
		With("redis", &fakePinger{}, false)
	w, body := serveReady(t, h)
	if w.Code != http.StatusOK || body.Status != StatusOK {
		t.Fatalf("code=%d status=%q", w.Code, body.Status)
	}
	for _, name := range []string{"postgres", "redis"} {
		c, ok := body.Checks[name]
		if !ok || c.Status != StatusOK || c.LatencyMS == nil {
			t.Fatalf("check %s = %+v", name, c)
		}
	}
	if !body.Checks["postgres"].Critical || body.Checks["redis"].Critical {
		t.Fatalf("critical flags wrong: %+v", body.Checks)
	}
}

func TestReadinessCriticalFailureIs503WithoutErrorText(t *testing.T) {
	var logs bytes.Buffer
	secret := "dial tcp db.internal:5432: password authentication failed"
	h := NewHandler(time.Second).
		WithLogger(slog.New(slog.NewTextHandler(&logs, nil))).
		With("postgres", &fakePinger{err: errors.New(secret)}, true).
		With("redis", &fakePinger{}, false)
	w, body := serveReady(t, h)
	if w.Code != http.StatusServiceUnavailable || body.Status != StatusUnavailable {
		t.Fatalf("code=%d status=%q", w.Code, body.Status)
	}
	if body.Checks["postgres"].Status != StatusUnavailable {
		t.Fatalf("postgres = %+v", body.Checks["postgres"])
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "db.internal") {
		t.Fatalf("body leaks error detail: %s", w.Body.String())
	}
	if !strings.Contains(logs.String(), "password authentication failed") || !strings.Contains(logs.String(), "dependency=postgres") {
		t.Fatalf("log missing detail: %s", logs.String())
	}
}

func TestReadinessNonCriticalFailureIsDegraded(t *testing.T) {
	h := NewHandler(time.Second).
		WithLogger(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))).
		With("postgres", &fakePinger{}, true).
		With("redis", &fakePinger{err: errors.New("redis down")}, false)
	w, body := serveReady(t, h)
	if w.Code != http.StatusOK || body.Status != StatusDegraded {
		t.Fatalf("code=%d status=%q", w.Code, body.Status)
	}
	if body.Checks["redis"].Status != StatusUnavailable || body.Checks["redis"].LatencyMS != nil {
		t.Fatalf("redis = %+v", body.Checks["redis"])
	}
}

func TestReadinessBlockingPingerTimesOut(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	timeout := 100 * time.Millisecond
	h := NewHandler(timeout).
		WithLogger(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))).
		With("postgres", PingFunc(func(context.Context) error { <-block; return nil }), true) // ignores ctx

	start := time.Now()
	w, body := serveReady(t, h)
	elapsed := time.Since(start)
	if w.Code != http.StatusServiceUnavailable || body.Status != StatusUnavailable {
		t.Fatalf("code=%d status=%q", w.Code, body.Status)
	}
	if elapsed > timeout+500*time.Millisecond {
		t.Fatalf("readiness took %v", elapsed)
	}
}

func TestReadinessNilDependenciesSkipped(t *testing.T) {
	var typedNil *fakePinger
	var nilFunc PingFunc
	h := NewHandler(0).With("a", nil, true).With("b", typedNil, true).With("c", nilFunc, true)
	w, body := serveReady(t, h)
	if w.Code != http.StatusOK || body.Status != StatusOK || len(body.Checks) != 0 {
		t.Fatalf("code=%d body=%+v", w.Code, body)
	}
}

func TestLiveness(t *testing.T) {
	r := gin.New()
	RegisterRoutes(r.Group("/api/health"), NewHandler(0))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health/live", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "alive") {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}
