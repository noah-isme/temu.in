package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv            string        `env:"APP_ENV,notEmpty"`
	Port              int           `env:"PORT" envDefault:"8080"`
	DatabaseURL       string        `env:"DATABASE_URL,notEmpty"`
	RedisURL          string        `env:"REDIS_URL,notEmpty"`
	JWTSecret         string        `env:"JWT_SECRET,notEmpty"`
	AccessTokenTTL    time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL   time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"`
	SendGridAPIKey    string        `env:"SENDGRID_API_KEY"`
	MidtransServerKey string        `env:"MIDTRANS_SERVER_KEY"`
	MidtransClientKey string        `env:"MIDTRANS_CLIENT_KEY"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	return cfg, nil
}
