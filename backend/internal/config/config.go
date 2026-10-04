package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv                         string        `env:"APP_ENV,notEmpty"`
	LogLevel                       string        `env:"LOG_LEVEL" envDefault:"info"`
	Port                           int           `env:"PORT" envDefault:"8080"`
	DatabaseURL                    string        `env:"DATABASE_URL,notEmpty"`
	RedisURL                       string        `env:"REDIS_URL,notEmpty"`
	JWTSecret                      string        `env:"JWT_SECRET,notEmpty"`
	AccessTokenTTL                 time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL                time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"`
	SendGridAPIKey                 string        `env:"SENDGRID_API_KEY"`
	SendGridFromEmail              string        `env:"SENDGRID_FROM_EMAIL" envDefault:"no-reply@temu.in"`
	SendGridBookingTemplateID      string        `env:"SENDGRID_BOOKING_TEMPLATE_ID"`
	SendGridCancellationTemplateID string        `env:"SENDGRID_CANCELLATION_TEMPLATE_ID"`
	SendGridReminderTemplateID     string        `env:"SENDGRID_REMINDER_TEMPLATE_ID"`
	SendGridReceiptTemplateID      string        `env:"SENDGRID_RECEIPT_TEMPLATE_ID"`
	SendGridAuthTemplateID         string        `env:"SENDGRID_AUTH_TEMPLATE_ID"`
	MidtransServerKey              string        `env:"MIDTRANS_SERVER_KEY"`
	MidtransClientKey              string        `env:"MIDTRANS_CLIENT_KEY"`
	MidtransBaseURL                string        `env:"MIDTRANS_BASE_URL" envDefault:"https://api.sandbox.midtrans.com"`
	AllowedOrigins                 []string      `env:"ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:5173,http://localhost:3000"`
	RateLimitRequests              int           `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	FrontendBaseURL                string        `env:"FRONTEND_BASE_URL" envDefault:"http://localhost:5173"`
	RateLimitWindow                time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	cfg.FrontendBaseURL = strings.TrimRight(strings.TrimSpace(cfg.FrontendBaseURL), "/")

	return cfg, nil
}
