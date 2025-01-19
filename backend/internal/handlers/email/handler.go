package email

import (
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
)

// Handler handles email-related HTTP requests
type Handler struct {
	emailService *email.Service
}

// NewHandler creates a new email handler
func NewHandler(emailService *email.Service) *Handler {
	return &Handler{
		emailService: emailService,
	}
}
