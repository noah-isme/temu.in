package review

import (
	"time"

	"temu.in/backend/internal/models"
)

// PublicReview is the review shape returned by every review endpoint. The
// reviewer is identified by display name only (never email or role).
type PublicReview struct {
	ID           string    `json:"id"`
	BookingID    uint      `json:"booking_id"`
	ProviderID   string    `json:"provider_id"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	ReviewerName string    `json:"reviewer_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewPublicReview maps a review model to its public DTO.
func NewPublicReview(r *models.Review) PublicReview {
	return PublicReview{
		ID:           r.ID,
		BookingID:    r.BookingID,
		ProviderID:   r.ProviderID,
		Rating:       r.Rating,
		Comment:      r.Comment,
		ReviewerName: r.User.Name,
		CreatedAt:    r.CreatedAt,
	}
}

func newPublicReviews(in []models.Review) []PublicReview {
	out := make([]PublicReview, 0, len(in))
	for i := range in {
		out = append(out, NewPublicReview(&in[i]))
	}
	return out
}
