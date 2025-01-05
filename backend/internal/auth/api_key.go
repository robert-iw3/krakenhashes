package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

const (
	KeyLength = 32 // 32 bytes = 64 hex chars
	KeyFile   = "agent.key"
	FilePerms = 0600 // Read/write for owner only
)

type KeyType string

const (
	KeyTypeAgent KeyType = "agent"
	KeyTypeUser  KeyType = "user"
)

// GenerateAPIKey generates a new 64-character hex API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SaveAgentKey saves the agent's API key and ID to a file with restricted permissions
func SaveAgentKey(configDir, apiKey, agentID string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	keyPath := filepath.Join(configDir, KeyFile)
	data := []byte(fmt.Sprintf("AGENT_ID=%s\nAPI_KEY=%s\n", agentID, apiKey))

	if err := os.WriteFile(keyPath, data, FilePerms); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	debug.Info("Saved agent key file to: %s", keyPath)
	return nil
}

// LoadAgentKey loads the agent's API key and ID from the key file
func LoadAgentKey(configDir string) (string, string, error) {
	keyPath := filepath.Join(configDir, KeyFile)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the key file content
	var agentID, apiKey string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "AGENT_ID=") {
			agentID = strings.TrimPrefix(line, "AGENT_ID=")
		} else if strings.HasPrefix(line, "API_KEY=") {
			apiKey = strings.TrimPrefix(line, "API_KEY=")
		}
	}

	if agentID == "" || apiKey == "" {
		return "", "", fmt.Errorf("invalid key file format")
	}

	return agentID, apiKey, nil
}

// ValidateAPIKey validates an API key against the database
func ValidateAPIKey(key string, keyType KeyType) (string, error) {
	// TODO: Implement database validation
	// This should check if the key exists and is active
	// Return the owner ID if valid
	return "", fmt.Errorf("not implemented")
}

// DisableAPIKey marks an API key as inactive
func DisableAPIKey(key string) error {
	// TODO: Implement database update
	// This should mark the key as inactive in the database
	return fmt.Errorf("not implemented")
}

// RotateAPIKey generates a new API key and invalidates the old one
func RotateAPIKey(oldKey string) (string, error) {
	// TODO: Implement key rotation
	// This should generate a new key and update the database
	return "", fmt.Errorf("not implemented")
}
