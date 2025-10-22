package audit

import (
    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
    "gorm.io/gorm"
)

type Repository struct{
    db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(a *models.AdminAudit) error { return r.db.Create(a).Error }

func (r *Repository) ListAll() ([]models.AdminAudit, error) {
    var out []models.AdminAudit
    if err := r.db.Order("created_at desc").Find(&out).Error; err != nil {
        return nil, err
    }
    return out, nil
}
