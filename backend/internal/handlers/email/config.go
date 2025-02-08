package email

import (
	"encoding/json"
	"fmt"
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

// UpdateConfigRequest represents the request body for updating email configuration
type UpdateConfigRequest struct {
	Config    emailtypes.Config `json:"config"`
	TestEmail string            `json:"test_email,omitempty"`
	TestOnly  bool              `json:"test_only"`
}

// UpdateConfig handles POST/PUT /api/email/config
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If test only, just test the configuration without saving
	if req.TestOnly {
		if err := h.emailService.TestConnectionWithConfig(ctx, &req.Config, req.TestEmail); err != nil {
			debug.Error("email configuration test failed: %v", err)
			http.Error(w, fmt.Sprintf("Email configuration test failed: %v", err), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Save the configuration
	if err := h.emailService.ConfigureProvider(ctx, &req.Config); err != nil {
		switch err {
		case email.ErrInvalidProvider:
			http.Error(w, "Invalid email provider", http.StatusBadRequest)
		default:
			debug.Error("failed to configure email provider: %v", err)
			http.Error(w, "Failed to configure email provider", http.StatusInternalServerError)
		}
		return
	}

	// If test email is provided, test the saved configuration
	if req.TestEmail != "" {
		if err := h.emailService.TestConnection(ctx, req.TestEmail); err != nil {
			debug.Error("email configuration test failed after save: %v", err)
			http.Error(w, fmt.Sprintf("Email configuration saved but test failed: %v", err), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// TestConfig handles POST /api/admin/email/test
func (h *Handler) TestConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var testReq struct {
		TestEmail string             `json:"test_email"`
		TestOnly  bool               `json:"test_only"`
		Config    *emailtypes.Config `json:"config,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&testReq); err != nil {
		debug.Error("failed to decode test request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if testReq.TestEmail == "" {
		http.Error(w, "Test email address is required", http.StatusBadRequest)
		return
	}

	// If config is provided, test with that config
	if testReq.Config != nil {
		if err := h.emailService.TestConnectionWithConfig(ctx, testReq.Config, testReq.TestEmail); err != nil {
			debug.Error("email configuration test failed: %v", err)
			http.Error(w, fmt.Sprintf("Email configuration test failed: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Test using the active configuration
		if err := h.emailService.TestConnection(ctx, testReq.TestEmail); err != nil {
			debug.Error("email configuration test failed: %v", err)
			http.Error(w, fmt.Sprintf("Email configuration test failed: %v", err), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
