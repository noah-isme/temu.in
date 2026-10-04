package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestContextRequestIDAddedToRecords(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf, "test", "info")

	l.InfoContext(WithRequestID(context.Background(), "abc"), "x")
	if !strings.Contains(buf.String(), "request_id=abc") {
		t.Fatalf("expected request_id=abc in %q", buf.String())
	}

	buf.Reset()
	l.InfoContext(context.Background(), "x")
	if strings.Contains(buf.String(), "request_id") {
		t.Fatalf("request_id should be absent, got %q", buf.String())
	}

	buf.Reset()
	l.With("component", "c").InfoContext(WithRequestID(context.Background(), "def"), "y")
	if !strings.Contains(buf.String(), "request_id=def") || !strings.Contains(buf.String(), "component=c") {
		t.Fatalf("derived logger lost request_id or attrs: %q", buf.String())
	}
}

func TestRequestIDFromContextEmpty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("got %q", got)
	}
}
