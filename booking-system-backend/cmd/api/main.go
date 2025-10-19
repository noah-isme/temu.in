package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/temu-in/temu.in/booking-system-backend/internal/config"
	"github.com/temu-in/temu.in/booking-system-backend/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if err := server.New(cfg).Run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
