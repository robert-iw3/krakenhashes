package agent

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/auth"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
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

	// Use the LoadAgentKey function to get the agent ID from agent.key
	_, agentIDStr, err := auth.LoadAgentKey(configDir)
	if err != nil {
		debug.Error("Failed to load agent ID from agent.key: %v", err)
		return 0, fmt.Errorf("failed to load agent ID from agent.key: %w", err)
	}

	id, err := strconv.Atoi(agentIDStr)
	if err != nil {
		debug.Error("Failed to parse agent ID: %v", err)
		return 0, fmt.Errorf("failed to parse agent ID: %w", err)
	}

	return id, nil
}
