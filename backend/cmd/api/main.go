package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"temu.in/backend/internal/config"
	"temu.in/backend/internal/logging"
	"temu.in/backend/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		// APP_ENV is unknown until config loads, so fall back to JSON.
		logging.New("", "info").Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.SetDefault(logging.New(cfg.AppEnv, cfg.LogLevel))

	if err := server.New(cfg).Run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
