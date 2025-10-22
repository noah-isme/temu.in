package user

import (
    "errors"

    "gorm.io/gorm"
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
)

type Repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Create(u *models.User) error {
    return r.db.Create(u).Error
}

func (r *Repository) FindByEmail(email string) (*models.User, error) {
    var u models.User
    if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &u, nil
}

func (r *Repository) FindByID(id uint) (*models.User, error) {
    var u models.User
    if err := r.db.First(&u, id).Error; err != nil {
        return nil, err
    }
    return &u, nil
}
