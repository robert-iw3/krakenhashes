package settings

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/httputil"
)

// RetentionSettingsHandler handles API requests related to data retention settings.
type RetentionSettingsHandler struct {
	repo *repository.ClientSettingsRepository
}

// NewRetentionSettingsHandler creates a new handler instance.
func NewRetentionSettingsHandler(r *repository.ClientSettingsRepository) *RetentionSettingsHandler {
	return &RetentionSettingsHandler{repo: r}
}

// GetDefaultRetention godoc
// @Summary Get default data retention setting
// @Description Retrieves the system-wide default data retention period in months.
// @Tags Admin Settings
// @Produce json
// @Success 200 {object} httputil.SuccessResponse{data=models.ClientSetting}
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/settings/retention [get]
// @Security ApiKeyAuth
func (h *RetentionSettingsHandler) GetDefaultRetention(w http.ResponseWriter, r *http.Request) {
	setting, err := h.repo.GetSetting(r.Context(), "default_data_retention_months")
	if err != nil {
		debug.Error("Failed to get default client retention setting: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve default client retention setting")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"data": setting})
}

// UpdateDefaultRetention godoc
// @Summary Update default data retention setting
// @Description Sets the system-wide default data retention period in months. 0 means keep forever.
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param setting body models.ClientSetting true "Setting object with key='default_data_retention_months' and new value (as string)"
// @Success 200 {object} httputil.SuccessResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/settings/retention [put]
// @Security ApiKeyAuth
func (h *RetentionSettingsHandler) UpdateDefaultRetention(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate input - must be a non-negative integer string
	months, err := strconv.Atoi(payload.Value)
	if err != nil || months < 0 {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid retention value: must be a non-negative integer string")
		return
	}

	valueStr := strconv.Itoa(months) // Ensure canonical string representation
	err = h.repo.SetSetting(r.Context(), "default_data_retention_months", &valueStr)
	if err != nil {
		debug.Error("Failed to update default client retention setting: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update default client retention setting")
		return
	}

	debug.Info("Default client data retention updated to %d months", months)
	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Default retention setting updated successfully"})
}
