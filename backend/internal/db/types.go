package db

import (
	"errors"

	"github.com/google/uuid"
)

// Error definitions
var (
	ErrNotFound = errors.New("record not found")
)

// User represents a user in the system
type User struct {
	ID         uuid.UUID `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	MFAEnabled bool      `json:"mfa_enabled"`
	MFAType    string    `json:"mfa_type"`
	MFASecret  string    `json:"-"` // Never expose in JSON
}
