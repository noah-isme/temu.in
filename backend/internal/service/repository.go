package service

import (
	"gorm.io/gorm"
	"temu.in/backend/internal/models"
)

type ServiceFilter struct {
	Name       string
	MinPrice   *float64
	MaxPrice   *float64
	ProviderID string
	IsActive   *bool
	Category   string
}

type Repository interface {
	Create(service *models.Service) error
	FindByID(id string) (*models.Service, error)
	FindByProviderID(providerID string) ([]models.Service, error)
	Update(service *models.Service) error
	Delete(id string) error
	ListByProvider(providerID string, activeOnly bool) ([]models.Service, error)
	Search(filter ServiceFilter) ([]models.Service, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(service *models.Service) error {
	return r.db.Create(service).Error
}

func (r *repository) FindByID(id string) (*models.Service, error) {
	var service models.Service
	err := r.db.Preload("Provider").First(&service, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func (r *repository) FindByProviderID(providerID string) ([]models.Service, error) {
	var services []models.Service
	err := r.db.Where("provider_id = ?", providerID).Find(&services).Error
	return services, err
}

func (r *repository) Update(service *models.Service) error {
	return r.db.Save(service).Error
}

func (r *repository) Delete(id string) error {
	return r.db.Delete(&models.Service{}, "id = ?", id).Error
}

func (r *repository) ListByProvider(providerID string, activeOnly bool) ([]models.Service, error) {
	var services []models.Service
	query := r.db.Where("provider_id = ?", providerID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("created_at DESC").Find(&services).Error
	return services, err
}

func (r *repository) Search(filter ServiceFilter) ([]models.Service, error) {
	var services []models.Service
	query := r.db.Model(&models.Service{}).Preload("Provider")

	if filter.ProviderID != "" {
		query = query.Where("provider_id = ?", filter.ProviderID)
	}
	if filter.Name != "" {
		// Use ILIKE for case-insensitive search
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.MinPrice != nil {
		query = query.Where("price >= ?", *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		query = query.Where("price <= ?", *filter.MaxPrice)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	err := query.Order("created_at DESC").Find(&services).Error
	return services, err
}
