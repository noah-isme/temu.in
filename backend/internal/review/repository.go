package review

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

func (r *Repository) Create(review *models.Review) error {
	return r.db.Create(review).Error
}

func (r *Repository) ListByProvider(providerID string, limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Where("provider_id = ?", providerID).
		Preload("User"). // To show who reviewed
		Limit(limit).
		Offset(offset).
		Order("created_at desc").
		Find(&reviews).Error
	return reviews, err
}

func (r *Repository) ListByUser(userID uint, limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Where("user_id = ?", userID).
		Preload("Provider").
		Limit(limit).
		Offset(offset).
		Order("created_at desc").
		Find(&reviews).Error
	return reviews, err
}

func (r *Repository) FindByBookingID(bookingID uint) (*models.Review, error) {
	var review models.Review
	err := r.db.Where("booking_id = ?", bookingID).First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// Stats aggregation
type ProviderReviewStats struct {
	AverageRating float64
	TotalReviews  int64
}

func (r *Repository) GetProviderStats(providerID string) (*ProviderReviewStats, error) {
	var stats ProviderReviewStats
	// Using SQL aggregation
	row := r.db.Model(&models.Review{}).
		Where("provider_id = ?", providerID).
		Select("COALESCE(AVG(rating), 0), COUNT(*)").
		Row()

	err := row.Scan(&stats.AverageRating, &stats.TotalReviews)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
