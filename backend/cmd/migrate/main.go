package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"temu.in/backend/internal/config"
	"temu.in/backend/internal/database"
	"temu.in/backend/internal/logging"
	"temu.in/backend/internal/seeder"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		// APP_ENV is unknown until config loads, so fall back to JSON.
		logging.New("", "info").Error("load config", "error", err)
		os.Exit(1)
	}
	slog.SetDefault(logging.New(cfg.AppEnv, cfg.LogLevel))

	if err := run(cfg); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied and admin seeding completed")
}

func run(cfg *config.Config) error {
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := seeder.SeedAdmin(db.DB); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	return nil
}
