package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func accessLogEngine(buf *bytes.Buffer) *gin.Engine {
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	r := gin.New()
	r.Use(RequestID(), AccessLog(logger))
	r.GET("/items/:id", func(c *gin.Context) { c.String(http.StatusOK, "hello") })
	r.GET("/bad", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	r.GET("/boom", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
	r.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/health/live", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func serve(r *gin.Engine, path, requestID string) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("decode %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestAccessLogOneLineWithStatusAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	r := accessLogEngine(&buf)
	serve(r, "/items/42", "req-1")

	lines := decodeLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("expected one access-log line, got %d: %s", len(lines), buf.String())
	}
	l := lines[0]
	if l["level"] != "INFO" || l["status"] != float64(200) || l["request_id"] != "req-1" {
		t.Fatalf("unexpected line %v", l)
	}
	if l["path"] != "/items/:id" || l["method"] != "GET" || l["bytes"] != float64(5) {
		t.Fatalf("unexpected line %v", l)
	}
	for _, k := range []string{"latency_ms", "client_ip", "user_agent"} {
		if _, ok := l[k]; !ok {
			t.Fatalf("missing %s in %v", k, l)
		}
	}
}

func TestAccessLogLevelsByStatus(t *testing.T) {
	var buf bytes.Buffer
	r := accessLogEngine(&buf)
	serve(r, "/bad", "")
	serve(r, "/boom", "")
	serve(r, "/unknown-route", "")

	lines := decodeLines(t, &buf)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0]["level"] != "WARN" || lines[1]["level"] != "ERROR" {
		t.Fatalf("levels = %v, %v", lines[0]["level"], lines[1]["level"])
	}
	if lines[2]["path"] != "/unknown-route" {
		t.Fatalf("unmatched route should log URL path, got %v", lines[2]["path"])
	}
}

func TestAccessLogSkipsHealthProbes(t *testing.T) {
	var buf bytes.Buffer
	r := accessLogEngine(&buf)
	serve(r, "/health/ready", "")
	serve(r, "/api/health/live", "")
	if buf.Len() != 0 {
		t.Fatalf("health probes should not be logged: %s", buf.String())
	}
}
