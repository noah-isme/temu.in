package payment

import (
    "gorm.io/gorm"

    "temu.in/backend/internal/models"
)

type Repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Create(p *models.Payment) error {
    return r.db.Create(p).Error
}

func (r *Repository) Update(p *models.Payment) error {
    return r.db.Save(p).Error
}

func (r *Repository) FindByIntent(intent string) (*models.Payment, error) {
    var p models.Payment
    if err := r.db.Where("intent_id = ?", intent).First(&p).Error; err != nil {
        return nil, err
    }
    return &p, nil
}

func (r *Repository) FindByClientSecret(cs string) (*models.Payment, error) {
    var p models.Payment
    if err := r.db.Where("client_secret = ?", cs).First(&p).Error; err != nil {
        return nil, err
    }
    return &p, nil
}

// DB exposes underlying gorm DB for limited internal use
func (r *Repository) DB() *gorm.DB { return r.db }
