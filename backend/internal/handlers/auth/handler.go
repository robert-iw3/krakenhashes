package auth

import (
	"context"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
)

// EmailService defines the interface for email operations needed by auth handlers
type EmailService interface {
	SendMFACode(ctx context.Context, to string, code string) error
}

// Handler handles authentication-related requests
type Handler struct {
	db           *db.DB
	emailService EmailService
}

// NewHandler creates a new auth handler
func NewHandler(db *db.DB, emailService EmailService) *Handler {
	return &Handler{
		db:           db,
		emailService: emailService,
	}
}

// NewHandlerWithEmailService creates a new auth handler with the concrete email service
// This is a convenience function for production code
func NewHandlerWithEmailService(db *db.DB, emailService *email.Service) *Handler {
	return NewHandler(db, emailService)
}
