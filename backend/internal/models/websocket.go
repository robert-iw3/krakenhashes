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

	// Download progress message types
	WSTypeDownloadProgress WSMessageType = "download_progress"
	WSTypeDownloadComplete WSMessageType = "download_complete"
	WSTypeDownloadFailed   WSMessageType = "download_failed"

	// Configuration message types
	WSTypeConfigUpdate WSMessageType = "config_update"
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

// DownloadProgressPayload represents progress information for a file download
type DownloadProgressPayload struct {
	FileName         string  `json:"file_name"`
	FileType         string  `json:"file_type"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	TotalBytes       int64   `json:"total_bytes"`
	PercentComplete  float64 `json:"percent_complete"`
	DownloadSpeed    string  `json:"download_speed,omitempty"`
	ETASeconds       int     `json:"eta_seconds,omitempty"`
}

// DownloadCompletePayload represents completion information for a file download
type DownloadCompletePayload struct {
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	TotalBytes   int64  `json:"total_bytes"`
	MD5Hash      string `json:"md5_hash"`
	DownloadTime int    `json:"download_time_seconds"`
}

// DownloadFailedPayload represents failure information for a file download
type DownloadFailedPayload struct {
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	Error        string `json:"error"`
	RetryAttempt int    `json:"retry_attempt,omitempty"`
	WillRetry    bool   `json:"will_retry"`
}
