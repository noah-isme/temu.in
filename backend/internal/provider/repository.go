package provider

import (
	"gorm.io/gorm"
	"temu.in/backend/internal/models"
)

type Repository interface {
	Create(provider *models.ServiceProvider) error
	FindByID(id string) (*models.ServiceProvider, error)
	FindByUserID(userID string) (*models.ServiceProvider, error)
	Update(provider *models.ServiceProvider) error
	Delete(id string) error
	List(filters ProviderFilters) ([]models.ServiceProvider, int64, error)
}

type ProviderFilters struct {
	BusinessType string `json:"business_type"`
	Search       string `json:"search"`
	IsActive     *bool  `json:"is_active"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(provider *models.ServiceProvider) error {
	return r.db.Create(provider).Error
}

func (r *repository) FindByID(id string) (*models.ServiceProvider, error) {
	var provider models.ServiceProvider
	err := r.db.Preload("User").First(&provider, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *repository) FindByUserID(userID string) (*models.ServiceProvider, error) {
	var provider models.ServiceProvider
	err := r.db.Preload("User").Where("user_id = ?", userID).First(&provider).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *repository) Update(provider *models.ServiceProvider) error {
	return r.db.Save(provider).Error
}

func (r *repository) Delete(id string) error {
	return r.db.Delete(&models.ServiceProvider{}, "id = ?", id).Error
}

func (r *repository) List(filters ProviderFilters) ([]models.ServiceProvider, int64, error) {
	var providers []models.ServiceProvider
	var total int64

	query := r.db.Model(&models.ServiceProvider{}).Preload("User")

	// Apply filters
	if filters.BusinessType != "" {
		query = query.Where("business_type = ?", filters.BusinessType)
	}
	if filters.Search != "" {
		query = query.Where("business_name ILIKE ? OR description ILIKE ?", "%"+filters.Search+"%", "%"+filters.Search+"%")
	}
	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}

	// Count total
	query.Count(&total)

	// Paginate
	offset := (filters.Page - 1) * filters.Limit
	err := query.Offset(offset).Limit(filters.Limit).Order("created_at DESC").Find(&providers).Error

	return providers, total, err
}
