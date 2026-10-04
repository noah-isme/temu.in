package config

import "testing"

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("REDIS_URL", "redis://x")
	t.Setenv("JWT_SECRET", "secret")
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("FRONTEND_BASE_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.FrontendBaseURL != "http://localhost:5173" {
		t.Errorf("FrontendBaseURL = %q, want default", cfg.FrontendBaseURL)
	}
}

func TestLoadTrimsFrontendBaseURLTrailingSlash(t *testing.T) {
	setRequired(t)
	t.Setenv("FRONTEND_BASE_URL", "https://app.example.test/")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FrontendBaseURL != "https://app.example.test" {
		t.Errorf("FrontendBaseURL = %q", cfg.FrontendBaseURL)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}
