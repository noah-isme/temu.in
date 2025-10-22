package main

import (
    "log"

    "github.com/joho/godotenv"
    "github.com/temu-in/temu.in/booking-system-backend/internal/config"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
    "github.com/temu-in/temu.in/booking-system-backend/internal/seeder"
)

func main() {
    _ = godotenv.Load()
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
    if err != nil {
        log.Fatalf("connect db: %v", err)
    }

    if err := db.AutoMigrate(&models.User{}); err != nil {
        log.Fatalf("migrate: %v", err)
    }

    if err := seeder.SeedAdmin(db); err != nil {
        log.Fatalf("seed admin: %v", err)
    }

    log.Println("migrations applied and admin seeding attempted")
}
