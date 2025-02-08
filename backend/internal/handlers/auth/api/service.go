package api

import (
	"context"
	"fmt"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// Service handles API key operations
type Service struct {
	db *db.DB
}

// NewService creates a new API key service
func NewService(db *db.DB) *Service {
	return &Service{db: db}
}

// ValidateKey validates an API key and returns the associated ID
func (s *Service) ValidateKey(ctx context.Context, key string, keyType KeyType) (string, error) {
	debug.Debug("Validating %s API key", keyType)

	// TODO: Implement validation logic using the database
	// This should:
	// 1. Check if the key exists
	// 2. Verify it's not expired or disabled
	// 3. Update last used timestamp
	// 4. Return the associated ID (user ID or agent ID)

	return "", fmt.Errorf("not implemented")
}

// CreateKey generates and stores a new API key
func (s *Service) CreateKey(ctx context.Context, ownerID string, keyType KeyType) (string, error) {
	debug.Debug("Creating new %s API key for owner %s", keyType, ownerID)

	key, err := GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// TODO: Implement storage logic
	// This should:
	// 1. Store the key in the database
	// 2. Associate it with the owner
	// 3. Set creation timestamp

	return key, fmt.Errorf("not implemented")
}

// RevokeKey disables an API key
func (s *Service) RevokeKey(ctx context.Context, key string) error {
	debug.Debug("Revoking API key")

	// TODO: Implement revocation logic
	// This should:
	// 1. Mark the key as disabled in the database
	// 2. Set revocation timestamp
	// 3. Log the revocation

	return fmt.Errorf("not implemented")
}

// RotateKey generates a new key and revokes the old one
func (s *Service) RotateKey(ctx context.Context, oldKey string) (string, error) {
	debug.Debug("Rotating API key")

	// TODO: Implement rotation logic
	// This should:
	// 1. Validate the old key
	// 2. Generate a new key
	// 3. Store the new key
	// 4. Revoke the old key
	// 5. Return the new key

	return "", fmt.Errorf("not implemented")
}
