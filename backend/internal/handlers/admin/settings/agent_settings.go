package settings

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// AgentSettingsHandler handles agent download settings requests
type AgentSettingsHandler struct {
	systemSettingsRepo *repository.SystemSettingsRepository
}

// NewAgentSettingsHandler creates a new agent settings handler
func NewAgentSettingsHandler(systemSettingsRepo *repository.SystemSettingsRepository) *AgentSettingsHandler {
	return &AgentSettingsHandler{
		systemSettingsRepo: systemSettingsRepo,
	}
}

// GetAgentDownloadSettings retrieves the current agent download settings
func (h *AgentSettingsHandler) GetAgentDownloadSettings(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Getting agent download settings")

	settings, err := h.systemSettingsRepo.GetAgentDownloadSettings(r.Context())
	if err != nil {
		debug.Error("Failed to get agent download settings: %v", err)
		http.Error(w, "Failed to get agent download settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// UpdateAgentDownloadSettings updates the agent download settings with validation
func (h *AgentSettingsHandler) UpdateAgentDownloadSettings(w http.ResponseWriter, r *http.Request) {
	debug.Info("Received request to update agent download settings")

	var settings models.AgentDownloadSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		debug.Error("Failed to decode agent download settings request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate settings
	if settings.MaxConcurrentDownloads < 1 || settings.MaxConcurrentDownloads > 10 {
		http.Error(w, "Maximum concurrent downloads must be between 1 and 10", http.StatusBadRequest)
		return
	}

	if settings.DownloadTimeoutMinutes < 1 || settings.DownloadTimeoutMinutes > 1440 { // Max 24 hours
		http.Error(w, "Download timeout must be between 1 and 1440 minutes (24 hours)", http.StatusBadRequest)
		return
	}

	if settings.DownloadRetryAttempts < 0 || settings.DownloadRetryAttempts > 10 {
		http.Error(w, "Download retry attempts must be between 0 and 10", http.StatusBadRequest)
		return
	}

	if settings.ProgressIntervalSeconds < 1 || settings.ProgressIntervalSeconds > 300 { // Max 5 minutes
		http.Error(w, "Progress interval must be between 1 and 300 seconds", http.StatusBadRequest)
		return
	}

	if settings.ChunkSizeMB < 1 || settings.ChunkSizeMB > 100 {
		http.Error(w, "Chunk size must be between 1 and 100 MB", http.StatusBadRequest)
		return
	}

	// Update settings in database
	if err := h.systemSettingsRepo.UpdateAgentDownloadSettings(r.Context(), &settings); err != nil {
		debug.Error("Failed to update agent download settings: %v", err)
		http.Error(w, "Failed to update agent download settings", http.StatusInternalServerError)
		return
	}

	debug.Info("Successfully updated agent download settings")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Agent download settings updated successfully",
		"settings": settings,
	})
}