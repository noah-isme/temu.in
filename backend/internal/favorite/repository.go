package favorite

import (
	"gorm.io/gorm"
	"temu.in/backend/internal/models"
)

type Repository interface {
	Create(fav *models.Favorite) error
	Delete(userID uint, favID string) error
	DeleteByTarget(userID uint, serviceID *string, providerID *string) error
	ListByUser(userID uint) ([]models.Favorite, error)
	Exists(userID uint, serviceID *string, providerID *string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(fav *models.Favorite) error {
	return r.db.Create(fav).Error
}

func (r *repository) Delete(userID uint, favID string) error {
	return r.db.Where("user_id = ? AND id = ?", userID, favID).Delete(&models.Favorite{}).Error
}

func (r *repository) DeleteByTarget(userID uint, serviceID *string, providerID *string) error {
	query := r.db.Where("user_id = ?", userID)
	if serviceID != nil {
		query = query.Where("service_id = ?", *serviceID)
	}
	if providerID != nil {
		query = query.Where("provider_id = ?", *providerID)
	}
	return query.Delete(&models.Favorite{}).Error
}

func (r *repository) ListByUser(userID uint) ([]models.Favorite, error) {
	var favs []models.Favorite
	err := r.db.Preload("Service").Preload("Provider").Where("user_id = ?", userID).Find(&favs).Error
	return favs, err
}

func (r *repository) Exists(userID uint, serviceID *string, providerID *string) (bool, error) {
	var count int64
	query := r.db.Model(&models.Favorite{}).Where("user_id = ?", userID)
	if serviceID != nil {
		query = query.Where("service_id = ?", *serviceID)
	}
	if providerID != nil {
		query = query.Where("provider_id = ?", *providerID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}
