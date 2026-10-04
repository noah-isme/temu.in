package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestIsDevEnv(t *testing.T) {
	for _, env := range []string{"development", "dev", "local", "test", "DEV"} {
		if !IsDevEnv(env) {
			t.Errorf("IsDevEnv(%q) = false, want true", env)
		}
	}
	for _, env := range []string{"", "production", "staging"} {
		if IsDevEnv(env) {
			t.Errorf("IsDevEnv(%q) = true, want false", env)
		}
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"":      slog.LevelInfo,
		"bogus": slog.LevelInfo,
	}
	for in, want := range cases {
		if got := ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewHandlerTypeByEnv(t *testing.T) {
	var buf bytes.Buffer
	NewWithWriter(&buf, "development", "info").Info("hello", "k", "v")
	line := buf.String()
	if !strings.Contains(line, "msg=hello") || strings.HasPrefix(line, "{") {
		t.Fatalf("development should log text, got %q", line)
	}

	buf.Reset()
	NewWithWriter(&buf, "production", "info").Info("hello", "k", "v")
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("production should log JSON, got %q: %v", buf.String(), err)
	}
	if m["msg"] != "hello" || m["k"] != "v" {
		t.Fatalf("unexpected JSON record %v", m)
	}
	if _, ok := m["source"]; ok {
		t.Fatal("AddSource should be disabled")
	}
}

func TestNewRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf, "production", "info")
	l.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("debug should be suppressed at info, got %q", buf.String())
	}
	l = NewWithWriter(&buf, "production", "debug")
	l.Debug("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Fatalf("debug should be enabled, got %q", buf.String())
	}
}
