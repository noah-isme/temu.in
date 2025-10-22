package seeder

import (
    "os"
    "log"

    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"

    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
)

func SeedAdmin(db *gorm.DB) error {
    if os.Getenv("ADMIN_SEED") != "true" {
        return nil
    }

    email := os.Getenv("ADMIN_EMAIL")
    password := os.Getenv("ADMIN_PASSWORD")
    if email == "" || password == "" {
        log.Println("ADMIN_SEED=true but ADMIN_EMAIL/ADMIN_PASSWORD not set; skipping admin seed")
        return nil
    }

    var u models.User
    if err := db.Where("email = ?", email).First(&u).Error; err == nil {
        // user exists
        return nil
    }

    pw, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    admin := &models.User{Email: email, Password: string(pw), Name: "Admin", Role: "admin"}
    return db.Create(admin).Error
}
