package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

const (
	KeyFile   = "agent.key"
	FilePerms = 0600 // Read/write for owner only
)

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

	return apiKey, agentID, nil
}
