package review

import (
	"errors"
	"fmt"
	"time"

	"temu.in/backend/internal/booking"
	"temu.in/backend/internal/models"
)

type Service struct {
	repo        *Repository
	bookingRepo *booking.Repository
}

func NewService(repo *Repository, bookingRepo *booking.Repository) *Service {
	return &Service{
		repo:        repo,
		bookingRepo: bookingRepo,
	}
}

type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

func (s *Service) CreateReview(userID uint, bookingID uint, req CreateReviewRequest) (*models.Review, error) {
	// 1. Get booking and validate ownership
	b, err := s.bookingRepo.FindByIDForUser(bookingID, userID)
	if err != nil {
		return nil, fmt.Errorf("booking not found or unauthorized")
	}

	// 2. Validate booking status
	if b.Status != "completed" {
		return nil, errors.New("can only review completed bookings")
	}

	// 3. Check if review already exists
	existing, err := s.repo.FindByBookingID(bookingID)
	if err == nil && existing != nil {
		return nil, errors.New("review already exists for this booking")
	}

	// 4. Create review
	review := &models.Review{
		BookingID:  bookingID,
		ProviderID: b.ProviderID,
		UserID:     userID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(review); err != nil {
		return nil, err
	}

	return review, nil
}

func (s *Service) ListProviderReviews(providerID string, limit, offset int) ([]models.Review, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	return s.repo.ListByProvider(providerID, limit, offset)
}

func (s *Service) ListUserReviews(userID uint, limit, offset int) ([]models.Review, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	return s.repo.ListByUser(userID, limit, offset)
}

func (s *Service) GetProviderStats(providerID string) (*ProviderReviewStats, error) {
	return s.repo.GetProviderStats(providerID)
}
