package models

// TimeSlot is the booking time contract shared by availability responses and
// booking requests. Keeping the date and both boundaries explicit prevents a
// client from sending an ambiguous, server-reparsed slot string.
type TimeSlot struct {
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}
