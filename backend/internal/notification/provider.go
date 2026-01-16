package notification

import (
	"log"
)

type ConsoleEmailProvider struct{}

func NewConsoleEmailProvider() *ConsoleEmailProvider {
	return &ConsoleEmailProvider{}
}

func (p *ConsoleEmailProvider) SendEmail(to []string, subject string, body string) error {
	// In production, integrate with SendGrid/SES
	// For dev, just log to console
	log.Printf("================ EMAIL NOTIFICATION ================")
	log.Printf("To: %v", to)
	log.Printf("Subject: %s", subject)
	log.Printf("Body: \n%s", body)
	log.Printf("====================================================")
	return nil
}

// Ensure interface implementation
var _ EmailProvider = (*ConsoleEmailProvider)(nil)
