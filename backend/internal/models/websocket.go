package models

import (
	"encoding/json"
	"time"
)

// WSMessageType represents different types of WebSocket messages
type WSMessageType string

const (
	WSTypeHardwareInfo WSMessageType = "hardware_info"
	WSTypeHeartbeat    WSMessageType = "heartbeat"
	WSTypeError        WSMessageType = "error"

	// File synchronization message types
	WSTypeFileSyncRequest  WSMessageType = "file_sync_request"
	WSTypeFileSyncResponse WSMessageType = "file_sync_response"
	WSTypeFileSyncCommand  WSMessageType = "file_sync_command"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type         WSMessageType   `json:"type"`
	HardwareInfo Hardware        `json:"hardware_info,omitempty"`
	OSInfo       json.RawMessage `json:"os_info,omitempty"`
	Error        string          `json:"error,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
}
