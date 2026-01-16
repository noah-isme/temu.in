package service

import (
	"errors"
	"fmt"

	"temu.in/backend/internal/models"
)

type Service interface {
	CreateService(providerID string, req *CreateServiceRequest) (*models.Service, error)
	GetService(id string) (*models.Service, error)
	GetServicesByProvider(providerID string, activeOnly bool) ([]models.Service, error)
	UpdateService(id string, providerID string, req *UpdateServiceRequest) (*models.Service, error)
	DeleteService(id string, providerID string) error
	SearchServices(filter ServiceFilter) ([]models.Service, error)
}

type CreateServiceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Duration    int     `json:"duration" binding:"required,min=1"`
	Price       float64 `json:"price" binding:"required,min=0"`
}

type UpdateServiceRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Duration    *int     `json:"duration" binding:"omitempty,min=1"`
	Price       *float64 `json:"price" binding:"omitempty,min=0"`
	IsActive    *bool    `json:"is_active"`
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateService(providerID string, req *CreateServiceRequest) (*models.Service, error) {
	service := &models.Service{
		ProviderID:  providerID,
		Name:        req.Name,
		Description: req.Description,
		Duration:    req.Duration,
		Price:       req.Price,
		IsActive:    true,
	}

	err := s.repo.Create(service)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return service, nil
}

func (s *service) GetService(id string) (*models.Service, error) {
	service, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}
	return service, nil
}

func (s *service) GetServicesByProvider(providerID string, activeOnly bool) ([]models.Service, error) {
	return s.repo.ListByProvider(providerID, activeOnly)
}

func (s *service) UpdateService(id string, providerID string, req *UpdateServiceRequest) (*models.Service, error) {
	service, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Check ownership
	if service.ProviderID != providerID {
		return nil, errors.New("unauthorized: service does not belong to provider")
	}

	// Update fields if provided
	if req.Name != nil {
		service.Name = *req.Name
	}
	if req.Description != nil {
		service.Description = *req.Description
	}
	if req.Duration != nil {
		service.Duration = *req.Duration
	}
	if req.Price != nil {
		service.Price = *req.Price
	}
	if req.IsActive != nil {
		service.IsActive = *req.IsActive
	}

	err = s.repo.Update(service)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	return service, nil
}

func (s *service) DeleteService(id string, providerID string) error {
	service, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	// Check ownership
	if service.ProviderID != providerID {
		return errors.New("unauthorized: service does not belong to provider")
	}

	return s.repo.Delete(id)
}

func (s *service) SearchServices(filter ServiceFilter) ([]models.Service, error) {
	return s.repo.Search(filter)
}
