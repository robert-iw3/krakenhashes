package models

// StatusUpdate represents a status update received, typically via WebSocket.
// This might come from agents reporting progress or cracks.
type StatusUpdate struct {
	HashlistID   int64  `json:"hashlist_id"`   // Changed to int64
	CrackedCount int    `json:"cracked_count"` // Number of *new* cracks in this update
	Status       string `json:"status"`        // Optional new status (e.g., "cracking", "paused", "completed")
	// Add other fields as needed, e.g., AgentID, ProgressPercent, ErrorMessage
}
