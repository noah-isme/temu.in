package main

import (
	"log"

	"github.com/joho/godotenv"

	"temu.in/backend/internal/config"
	"temu.in/backend/internal/database"
	"temu.in/backend/internal/seeder"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	if err := seeder.SeedAdmin(db.DB); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	log.Println("migrations applied and admin seeding completed")
}
