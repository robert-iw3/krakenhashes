package models

// AgentDownloadSettings represents the configuration for agent file downloads
type AgentDownloadSettings struct {
	MaxConcurrentDownloads      int `json:"max_concurrent_downloads"`
	DownloadTimeoutMinutes      int `json:"download_timeout_minutes"`
	DownloadRetryAttempts       int `json:"download_retry_attempts"`
	ProgressIntervalSeconds     int `json:"progress_interval_seconds"`
	ChunkSizeMB                 int `json:"chunk_size_mb"`
}

// GetDefaultAgentDownloadSettings returns the default download settings
func GetDefaultAgentDownloadSettings() AgentDownloadSettings {
	return AgentDownloadSettings{
		MaxConcurrentDownloads:      3,
		DownloadTimeoutMinutes:      60,
		DownloadRetryAttempts:       3,
		ProgressIntervalSeconds:     10,
		ChunkSizeMB:                 10,
	}
}