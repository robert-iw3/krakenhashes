package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ZerkerEOD/hashdom-backend/internal/config"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// RegistrationRequest represents the data sent by the agent during registration
type RegistrationRequest struct {
	ClaimCode string `json:"claim_code"`
	Hostname  string `json:"hostname"`
}

// RegistrationResponse represents the data sent back to the agent after successful registration
type RegistrationResponse struct {
	AgentID   int               `json:"agent_id"`
	APIKey    string            `json:"api_key"`
	Endpoints map[string]string `json:"endpoints"`
}

// RegistrationHandler handles agent registration and API key distribution
type RegistrationHandler struct {
	agentService *services.AgentService
	config       *config.Config
}

// NewRegistrationHandler creates a new instance of RegistrationHandler
func NewRegistrationHandler(agentService *services.AgentService, config *config.Config) *RegistrationHandler {
	return &RegistrationHandler{
		agentService: agentService,
		config:       config,
	}
}

// HandleRegistration handles agent registration requests
func (h *RegistrationHandler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	debug.Info("Received agent registration request")

	// Parse request body
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Failed to decode registration request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	debug.Debug("Registration request - Claim Code: %s, Hostname: %s", req.ClaimCode, req.Hostname)

	// Register agent
	agent, err := h.agentService.RegisterAgent(r.Context(), req.ClaimCode, req.Hostname)
	if err != nil {
		debug.Error("Agent registration failed: %v", err)
		http.Error(w, fmt.Sprintf("Registration failed: %v", err), http.StatusBadRequest)
		return
	}
	debug.Info("Successfully registered agent with ID: %d", agent.ID)

	// Prepare response
	resp := RegistrationResponse{
		AgentID: agent.ID,
		APIKey:  agent.APIKey.String,
		Endpoints: map[string]string{
			"websocket": h.config.GetWSEndpoint(),
			"api":       h.config.GetAPIEndpoint(),
		},
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		debug.Error("Failed to encode registration response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	debug.Info("Successfully sent registration response")
}
