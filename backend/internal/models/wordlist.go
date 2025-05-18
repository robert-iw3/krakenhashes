package models

import (
	"time"

	"github.com/google/uuid"
)

// WordlistType represents the type of wordlist
type WordlistType string

// Wordlist types
const (
	WordlistTypeGeneral     WordlistType = "general"
	WordlistTypeSpecialized WordlistType = "specialized"
	WordlistTypeTargeted    WordlistType = "targeted"
	WordlistTypeCustom      WordlistType = "custom"
)

// WordlistFormat represents the format of a wordlist
type WordlistFormat string

// Wordlist formats
const (
	WordlistFormatPlaintext  WordlistFormat = "plaintext"
	WordlistFormatCompressed WordlistFormat = "compressed"
)

// Wordlist represents the structure of the 'wordlists' table.
// Note: Add other fields from migration 000013 if needed for other contexts.
type Wordlist struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description"`
	WordlistType       string    `json:"wordlist_type"` // e.g., "dictionary", "password", "custom"
	Format             string    `json:"format"`        // e.g., "txt", "gz", "zip"
	FileName           string    `json:"file_name"`
	MD5Hash            string    `json:"md5_hash"`
	FileSize           int64     `json:"file_size" db:"file_size"`
	WordCount          int64     `json:"word_count"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	CreatedBy          uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          uuid.UUID `json:"updated_by,omitempty"`
	LastVerifiedAt     time.Time `json:"last_verified_at,omitempty"`
	VerificationStatus string    `json:"verification_status"` // e.g., "pending", "verified", "failed"
	Tags               []string  `json:"tags,omitempty"`
}

// WordlistBasic is a subset of Wordlist used for simple listings (e.g., form data).
type WordlistBasic struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// WordlistAddRequest represents a request to add a new wordlist
type WordlistAddRequest struct {
	Name         string   `json:"name" validate:"required"`
	Description  string   `json:"description"`
	WordlistType string   `json:"wordlist_type" validate:"required"`
	Format       string   `json:"format" validate:"required"`
	FileName     string   `json:"file_name" validate:"required"`
	MD5Hash      string   `json:"md5_hash" validate:"required"`
	FileSize     int64    `json:"file_size" validate:"required"`
	WordCount    int64    `json:"word_count"`
	Tags         []string `json:"tags"`
}

// WordlistUpdateRequest represents a request to update an existing wordlist
type WordlistUpdateRequest struct {
	Name         string   `json:"name" validate:"required"`
	Description  string   `json:"description"`
	WordlistType string   `json:"wordlist_type" validate:"required"`
	Format       string   `json:"format"`
	Tags         []string `json:"tags"`
}

// WordlistVerifyRequest represents a request to verify a wordlist
type WordlistVerifyRequest struct {
	Status    string `json:"status" validate:"required,oneof=pending verified failed"`
	WordCount *int64 `json:"word_count,omitempty"`
}

// WordlistTagRequest represents a request to add or remove a tag
type WordlistTagRequest struct {
	Tag string `json:"tag" validate:"required"`
}

// WordlistTag represents a tag associated with a wordlist
type WordlistTag struct {
	ID         int       `json:"id" db:"id"`
	WordlistID int       `json:"wordlist_id" db:"wordlist_id"`
	Tag        string    `json:"tag" db:"tag"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	CreatedBy  uuid.UUID `json:"created_by" db:"created_by"`
}

// WordlistAuditLog represents an entry in the wordlist audit log
type WordlistAuditLog struct {
	ID          int       `json:"id" db:"id"`
	WordlistID  int       `json:"wordlist_id" db:"wordlist_id"`
	Action      string    `json:"action" db:"action"`
	PerformedBy uuid.UUID `json:"performed_by" db:"performed_by"`
	PerformedAt time.Time `json:"performed_at" db:"performed_at"`
	Details     []byte    `json:"details" db:"details"`
}
