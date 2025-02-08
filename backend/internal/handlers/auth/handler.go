package auth

import (
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
)

// Handler handles authentication-related requests
type Handler struct {
	db           *db.DB
	emailService *email.Service
}

// NewHandler creates a new auth handler
func NewHandler(db *db.DB, emailService *email.Service) *Handler {
	return &Handler{
		db:           db,
		emailService: emailService,
	}
}
