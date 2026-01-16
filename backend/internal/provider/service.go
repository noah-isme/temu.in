package provider

import (
	"errors"
	"fmt"

	"temu.in/backend/internal/models"
)

type Service interface {
	CreateProvider(userID string, req *CreateProviderRequest) (*models.ServiceProvider, error)
	GetProvider(id string) (*models.ServiceProvider, error)
	GetProviderByUserID(userID string) (*models.ServiceProvider, error)
	UpdateProvider(id string, userID string, req *UpdateProviderRequest) (*models.ServiceProvider, error)
	DeleteProvider(id string, userID string) error
	ListProviders(filters ProviderFilters) ([]models.ServiceProvider, int64, error)
}

type CreateProviderRequest struct {
	BusinessName string   `json:"business_name" binding:"required"`
	BusinessType string   `json:"business_type" binding:"required,oneof=doctor salon consultant studio"`
	Description  string   `json:"description"`
	Address      string   `json:"address" binding:"required"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	PricingInfo  string   `json:"pricing_info"`
}

type UpdateProviderRequest struct {
	BusinessName *string  `json:"business_name"`
	BusinessType *string  `json:"business_type" binding:"omitempty,oneof=doctor salon consultant studio"`
	Description  *string  `json:"description"`
	Address      *string  `json:"address"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	PricingInfo  *string  `json:"pricing_info"`
	IsActive     *bool    `json:"is_active"`
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateProvider(userID string, req *CreateProviderRequest) (*models.ServiceProvider, error) {
	// Check if user already has a provider profile
	existing, err := s.repo.FindByUserID(userID)
	if err == nil && existing != nil {
		return nil, errors.New("user already has a provider profile")
	}

	provider := &models.ServiceProvider{
		UserID:       userID,
		BusinessName: req.BusinessName,
		BusinessType: req.BusinessType,
		Description:  req.Description,
		Address:      req.Address,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		PricingInfo:  req.PricingInfo,
		IsActive:     true,
	}

	err = s.repo.Create(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

func (s *service) GetProvider(id string) (*models.ServiceProvider, error) {
	provider, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

func (s *service) GetProviderByUserID(userID string) (*models.ServiceProvider, error) {
	provider, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

func (s *service) UpdateProvider(id string, userID string, req *UpdateProviderRequest) (*models.ServiceProvider, error) {
	provider, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Check ownership
	if provider.UserID != userID {
		return nil, errors.New("unauthorized: provider does not belong to user")
	}

	if req.BusinessName != nil {
		provider.BusinessName = *req.BusinessName
	}
	if req.BusinessType != nil {
		provider.BusinessType = *req.BusinessType
	}
	if req.Description != nil {
		provider.Description = *req.Description
	}
	if req.Address != nil {
		provider.Address = *req.Address
	}
	if req.Latitude != nil {
		provider.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		provider.Longitude = req.Longitude
	}
	if req.PricingInfo != nil {
		provider.PricingInfo = *req.PricingInfo
	}
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}

	err = s.repo.Update(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return provider, nil
}

func (s *service) DeleteProvider(id string, userID string) error {
	provider, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Check ownership
	if provider.UserID != userID {
		return errors.New("unauthorized: provider does not belong to user")
	}

	return s.repo.Delete(id)
}

func (s *service) ListProviders(filters ProviderFilters) ([]models.ServiceProvider, int64, error) {
	return s.repo.List(filters)
}
