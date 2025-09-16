package auth

import (
	"strconv"
	"strings"
)

// ParseAgentID extracts the numeric agent ID from the string format "agent_<id>"
func ParseAgentID(agentIDStr string) int {
	// Remove the "agent_" prefix if present
	idStr := strings.TrimPrefix(agentIDStr, "agent_")

	// Parse the ID as integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// Return 0 if parsing fails
		return 0
	}

	return id
}