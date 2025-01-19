package email

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	emailtypes "github.com/ZerkerEOD/krakenhashes/backend/pkg/email"
)

// GetConfig handles GET /api/email/config
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := h.emailService.GetConfig(ctx)
	if err == email.ErrConfigNotFound {
		http.Error(w, "Email configuration not found", http.StatusNotFound)
		return
	}
	if err != nil {
		debug.Error("failed to get email config: %v", err)
		http.Error(w, "Failed to get email configuration", http.StatusInternalServerError)
		return
	}

	// Redact sensitive information
	config.APIKey = "[REDACTED]"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateConfig handles POST/PUT /api/email/config
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var config emailtypes.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.emailService.ConfigureProvider(ctx, &config); err != nil {
		switch err {
		case email.ErrInvalidProvider:
			http.Error(w, "Invalid email provider", http.StatusBadRequest)
		default:
			debug.Error("failed to configure email provider: %v", err)
			http.Error(w, "Failed to configure email provider", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// TestConfig handles POST /api/email/test
func (h *Handler) TestConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var testConfig emailtypes.TestConfig
	if err := json.NewDecoder(r.Body).Decode(&testConfig); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if testConfig.TestEmail == "" {
		http.Error(w, "Test email address is required", http.StatusBadRequest)
		return
	}

	if err := h.emailService.TestConnection(ctx, testConfig.TestEmail); err != nil {
		debug.Error("email configuration test failed: %v", err)
		http.Error(w, "Email configuration test failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
