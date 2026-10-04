package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"temu.in/backend/internal/logging"
)

func init() { gin.SetMode(gin.TestMode) }

func requestIDEngine(seen *string, ctxSeen *string) *gin.Engine {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		*seen = RequestIDFromGin(c)
		*ctxSeen = logging.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})
	return r
}

func TestRequestIDEchoesSaneIncomingID(t *testing.T) {
	var seen, ctxSeen string
	r := requestIDEngine(&seen, &ctxSeen)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "abc-123_x.Y")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "abc-123_x.Y" {
		t.Fatalf("response header = %q", got)
	}
	if seen != "abc-123_x.Y" || ctxSeen != "abc-123_x.Y" {
		t.Fatalf("gin=%q ctx=%q", seen, ctxSeen)
	}
}

func TestRequestIDReplacesUnsafeOrMissingID(t *testing.T) {
	for name, header := range map[string]string{
		"missing":  "",
		"spaces":   "has spaces",
		"too long": strings.Repeat("a", 300),
		"newline":  "a\nb",
	} {
		t.Run(name, func(t *testing.T) {
			var seen, ctxSeen string
			r := requestIDEngine(&seen, &ctxSeen)
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if header != "" {
				req.Header.Set("X-Request-ID", header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			got := w.Header().Get("X-Request-ID")
			if _, err := uuid.Parse(got); err != nil {
				t.Fatalf("expected generated UUID, got %q", got)
			}
			if seen != got || ctxSeen != got {
				t.Fatalf("header=%q gin=%q ctx=%q", got, seen, ctxSeen)
			}
		})
	}
}

func TestRequestIDAccepts128Chars(t *testing.T) {
	var seen, ctxSeen string
	r := requestIDEngine(&seen, &ctxSeen)
	id := strings.Repeat("b", 128)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", id)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-ID") != id {
		t.Fatal("128-char id should be accepted")
	}
}
