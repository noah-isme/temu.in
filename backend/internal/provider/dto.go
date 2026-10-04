package provider

import (
	"time"

	"temu.in/backend/internal/models"
)

// PublicProvider is the provider shape returned by every provider endpoint.
// It exposes the owner's display name only; the owner's email, role and
// verification state never leave the API.
type PublicProvider struct {
	ID           string    `json:"id"`
	BusinessName string    `json:"business_name"`
	BusinessType string    `json:"business_type"`
	Description  string    `json:"description"`
	Address      string    `json:"address"`
	Latitude     *float64  `json:"latitude"`
	Longitude    *float64  `json:"longitude"`
	PricingInfo  string    `json:"pricing_info"`
	IsActive     bool      `json:"is_active"`
	OwnerName    string    `json:"owner_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewPublicProvider maps a provider model to its public DTO.
func NewPublicProvider(p *models.ServiceProvider) PublicProvider {
	return PublicProvider{
		ID:           p.ID,
		BusinessName: p.BusinessName,
		BusinessType: p.BusinessType,
		Description:  p.Description,
		Address:      p.Address,
		Latitude:     p.Latitude,
		Longitude:    p.Longitude,
		PricingInfo:  p.PricingInfo,
		IsActive:     p.IsActive,
		OwnerName:    p.User.Name,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func newPublicProviders(in []models.ServiceProvider) []PublicProvider {
	out := make([]PublicProvider, 0, len(in))
	for i := range in {
		out = append(out, NewPublicProvider(&in[i]))
	}
	return out
}
