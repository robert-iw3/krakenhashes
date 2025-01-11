package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
)

const (
	// File names for agent data
	idFile = "agent_id"
)

// Agent represents an agent instance
type Agent struct {
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	LastSeenAt    time.Time         `json:"last_seen_at"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Metadata      map[string]string `json:"metadata"`
	DownloadToken string            `json:"download_token"`
}

// GetAgentID retrieves the agent ID from the filesystem
func GetAgentID() (int, error) {
	configDir := config.GetConfigDir()
	idPath := filepath.Join(configDir, idFile)
	idBytes, err := os.ReadFile(idPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read agent ID: %w", err)
	}

	id, err := strconv.Atoi(string(idBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to parse agent ID: %w", err)
	}

	return id, nil
}
