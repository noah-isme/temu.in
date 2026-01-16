package favorite

import (
	"errors"

	"temu.in/backend/internal/models"
)

type Service interface {
	ToggleFavorite(userID uint, serviceID *string, providerID *string) (bool, error) // Returns true if added, false if removed
	ListFavorites(userID uint) ([]models.Favorite, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ToggleFavorite(userID uint, serviceID *string, providerID *string) (bool, error) {
	if serviceID == nil && providerID == nil {
		return false, errors.New("either serviceId or providerId must be provided")
	}

	exists, err := s.repo.Exists(userID, serviceID, providerID)
	if err != nil {
		return false, err
	}

	if exists {
		// Remove
		err = s.repo.DeleteByTarget(userID, serviceID, providerID)
		return false, err
	}

	// Add
	fav := &models.Favorite{
		UserID:     userID,
		ProviderID: providerID,
		ServiceID:  serviceID,
	}

	err = s.repo.Create(fav)
	return true, err
}

func (s *service) ListFavorites(userID uint) ([]models.Favorite, error) {
	return s.repo.ListByUser(userID)
}
