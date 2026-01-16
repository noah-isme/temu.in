package main

import (
    "log"

    "github.com/joho/godotenv"
    "temu.in/backend/internal/config"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "temu.in/backend/internal/models"
    "temu.in/backend/internal/seeder"
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

    if err := db.AutoMigrate(&models.User{}, &models.Booking{}, &models.Payment{}); err != nil {
        log.Fatalf("migrate: %v", err)
    }

    // Ensure payment_id column exists and has a FK to payments.id (idempotent)
    if err := db.Exec(`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS payment_id bigint NULL`).Error; err != nil {
        log.Fatalf("ensure payment_id column: %v", err)
    }
    // Add FK constraint if not exists (Postgres supports IF NOT EXISTS only for indexes, so use safe CREATE TABLE-like check)
    // We'll try to create constraint; if it errors because it exists, ignore.
    // Create foreign key constraint only if it doesn't already exist (Postgres)
    if err := db.Exec(`DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_bookings_payment') THEN
            ALTER TABLE bookings ADD CONSTRAINT fk_bookings_payment FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE SET NULL;
        END IF;
    END
    $$;`).Error; err != nil {
        log.Printf("warning creating fk constraint: %v", err)
    }

    if err := seeder.SeedAdmin(db); err != nil {
        log.Fatalf("seed admin: %v", err)
    }

    // Ensure user_id column exists and backfill existing bookings safely
    if err := db.Exec(`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS user_id bigint NULL`).Error; err != nil {
        log.Fatalf("ensure user_id column: %v", err)
    }

    // Backfill existing NULL user_id bookings to an admin user if one exists
    // This is idempotent: it only runs an update if admin exists and bookings.user_id IS NULL
    if err := db.Exec(`DO $$
    DECLARE
        admin_id bigint;
    BEGIN
        SELECT id INTO admin_id FROM users WHERE role = 'admin' LIMIT 1;
        IF admin_id IS NOT NULL THEN
            UPDATE bookings SET user_id = admin_id WHERE user_id IS NULL;
        END IF;
    END
    $$;`).Error; err != nil {
        log.Printf("warning backfilling user_id: %v", err)
    }

    // Add foreign key constraint if not exists
    if err := db.Exec(`DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_bookings_user') THEN
            ALTER TABLE bookings ADD CONSTRAINT fk_bookings_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
        END IF;
    END
    $$;`).Error; err != nil {
        log.Printf("warning creating fk constraint for user_id: %v", err)
    }

    log.Println("migrations applied and admin seeding attempted")
}
