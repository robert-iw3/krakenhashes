package email

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// UsageResponse represents the email usage statistics response
type UsageResponse struct {
	CurrentMonth struct {
		Count     int       `json:"count"`
		Limit     *int      `json:"limit,omitempty"`
		ResetDate time.Time `json:"reset_date"`
		Remaining *int      `json:"remaining,omitempty"`
		IsLimited bool      `json:"is_limited"`
	} `json:"current_month"`
	History []struct {
		MonthYear time.Time `json:"month_year"`
		Count     int       `json:"count"`
	} `json:"history"`
}

// GetUsage handles GET /api/email/usage
func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current configuration for limits
	config, err := h.emailService.GetConfig(ctx)
	if err != nil {
		debug.Error("failed to get email config: %v", err)
		http.Error(w, "Failed to get usage statistics", http.StatusInternalServerError)
		return
	}

	// TODO: Implement usage tracking in the service layer
	// For now, return a placeholder response
	response := UsageResponse{}
	response.CurrentMonth.Count = 0
	response.CurrentMonth.Limit = config.MonthlyLimit
	response.CurrentMonth.ResetDate = time.Now().AddDate(0, 1, 0).UTC()
	response.CurrentMonth.IsLimited = config.MonthlyLimit != nil

	if config.MonthlyLimit != nil {
		remaining := *config.MonthlyLimit - response.CurrentMonth.Count
		response.CurrentMonth.Remaining = &remaining
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
