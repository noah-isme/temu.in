package booking

import "temu.in/backend/internal/models"

const dtoDateLayout = "2006-01-02"

// ProviderBookingDTO is what a provider owner (or admin) sees for a booking
// on their dashboard. It deliberately carries no customer email, role, or
// admin notes: the customer is identified only by the name they entered.
type ProviderBookingDTO struct {
	ID                 uint    `json:"id"`
	CustomerName       string  `json:"customer_name"`
	ServiceID          *string `json:"service_id,omitempty"`
	ServiceName        string  `json:"service_name,omitempty"`
	BookingDate        string  `json:"booking_date"`
	StartTime          string  `json:"start_time"`
	EndTime            string  `json:"end_time"`
	Status             string  `json:"status"`
	PaymentStatus      string  `json:"payment_status"`
	CustomerNotes      string  `json:"customer_notes,omitempty"`
	CancellationReason string  `json:"cancellation_reason,omitempty"`
}

// CustomerBookingDTO is the customer-facing view of their own booking. Admin
// notes are never exposed. provider_name/service_name are present only when
// the booking was loaded with its associations (not on the create response).
type CustomerBookingDTO struct {
	ID                 uint    `json:"id"`
	ProviderID         string  `json:"provider_id"`
	ProviderName       string  `json:"provider_name,omitempty"`
	ServiceID          *string `json:"service_id,omitempty"`
	ServiceName        string  `json:"service_name,omitempty"`
	BookingDate        string  `json:"booking_date"`
	StartTime          string  `json:"start_time"`
	EndTime            string  `json:"end_time"`
	Status             string  `json:"status"`
	PaymentStatus      string  `json:"payment_status"`
	CustomerNotes      string  `json:"customer_notes,omitempty"`
	CancellationReason string  `json:"cancellation_reason,omitempty"`
	PaymentID          *uint   `json:"payment_id,omitempty"`
}

func serviceName(b *models.Booking) string {
	if b.Service == nil {
		return ""
	}
	return b.Service.Name
}

// NewProviderBookingDTO maps a booking to the provider-facing DTO.
func NewProviderBookingDTO(b *models.Booking) ProviderBookingDTO {
	return ProviderBookingDTO{
		ID:                 b.ID,
		CustomerName:       b.CustomerName,
		ServiceID:          b.ServiceID,
		ServiceName:        serviceName(b),
		BookingDate:        b.BookingDate.Format(dtoDateLayout),
		StartTime:          models.FormatClockHM(b.StartTime),
		EndTime:            models.FormatClockHM(b.EndTime),
		Status:             b.Status,
		PaymentStatus:      b.PaymentStatus,
		CustomerNotes:      b.CustomerNotes,
		CancellationReason: b.CancellationReason,
	}
}

// NewCustomerBookingDTO maps a booking to the customer-facing DTO.
func NewCustomerBookingDTO(b *models.Booking) CustomerBookingDTO {
	return CustomerBookingDTO{
		ID:                 b.ID,
		ProviderID:         b.ProviderID,
		ProviderName:       b.ServiceProvider.BusinessName,
		ServiceID:          b.ServiceID,
		ServiceName:        serviceName(b),
		BookingDate:        b.BookingDate.Format(dtoDateLayout),
		StartTime:          models.FormatClockHM(b.StartTime),
		EndTime:            models.FormatClockHM(b.EndTime),
		Status:             b.Status,
		PaymentStatus:      b.PaymentStatus,
		CustomerNotes:      b.CustomerNotes,
		CancellationReason: b.CancellationReason,
		PaymentID:          b.PaymentID,
	}
}

func newProviderBookingDTOs(in []models.Booking) []ProviderBookingDTO {
	out := make([]ProviderBookingDTO, 0, len(in))
	for i := range in {
		out = append(out, NewProviderBookingDTO(&in[i]))
	}
	return out
}

func newCustomerBookingDTOs(in []models.Booking) []CustomerBookingDTO {
	out := make([]CustomerBookingDTO, 0, len(in))
	for i := range in {
		out = append(out, NewCustomerBookingDTO(&in[i]))
	}
	return out
}
