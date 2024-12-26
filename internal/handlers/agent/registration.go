package agent

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// InitialRegistrationResponse represents the initial registration response
type InitialRegistrationResponse struct {
	AgentID       string            `json:"agent_id"`
	DownloadToken string            `json:"download_token"`
	Endpoints     map[string]string `json:"endpoints"`
}

// CertificateResponse represents the certificate download response
type CertificateResponse struct {
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
}

// RegistrationHandler handles agent registration
type RegistrationHandler struct {
	agentService *services.AgentService
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(agentService *services.AgentService) *RegistrationHandler {
	return &RegistrationHandler{
		agentService: agentService,
	}
}

// HandleRegistration handles the initial registration request
func (h *RegistrationHandler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Failed to decode registration request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Register agent
	agent, err := h.agentService.RegisterAgent(r.Context(), req.ClaimCode, req.Hostname)
	if err != nil {
		debug.Error("Failed to register agent: %v", err)
		http.Error(w, "Registration failed", http.StatusBadRequest)
		return
	}

	// Generate a temporary download token
	downloadToken, err := h.agentService.CreateDownloadToken(r.Context(), agent.ID)
	if err != nil {
		debug.Error("Failed to create download token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response with endpoints
	resp := InitialRegistrationResponse{
		AgentID:       agent.ID,
		DownloadToken: downloadToken,
		Endpoints: map[string]string{
			"cert":      "/api/agent/cert",
			"config":    "/api/agent/config",
			"websocket": "/ws/agent",
			"upload":    "/api/agent/upload",
			"download":  "/api/agent/download",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCertificateDownload handles the certificate download request
func (h *RegistrationHandler) HandleCertificateDownload(w http.ResponseWriter, r *http.Request) {
	// Extract download token from header
	downloadToken := r.Header.Get("X-Download-Token")
	if downloadToken == "" {
		debug.Error("No download token provided")
		http.Error(w, "Download token required", http.StatusUnauthorized)
		return
	}

	// Validate download token
	agentID, err := h.agentService.ValidateDownloadToken(r.Context(), downloadToken)
	if err != nil {
		debug.Error("Invalid download token: %v", err)
		http.Error(w, "Invalid download token", http.StatusUnauthorized)
		return
	}

	// Get agent's certificates
	agent, err := h.agentService.GetAgent(r.Context(), agentID)
	if err != nil {
		debug.Error("Failed to get agent: %v", err)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Prepare certificate response
	resp := CertificateResponse{
		Certificate:   agent.Certificate.String,
		PrivateKey:    agent.PrivateKey.String,
		CACertificate: "", // This will need to be provided by a separate service
	}

	// Invalidate download token after successful download
	if err := h.agentService.InvalidateDownloadToken(r.Context(), downloadToken); err != nil {
		debug.Warning("Failed to invalidate download token: %v", err)
		// Continue anyway as this isn't critical
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
