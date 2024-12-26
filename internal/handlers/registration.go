package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// RegistrationRequest represents the data received from an agent during registration
type RegistrationRequest struct {
	ClaimCode string `json:"claim_code"`
	Hostname  string `json:"hostname"`
}

// RegistrationResponse represents the data sent back to the agent after successful registration
type RegistrationResponse struct {
	AgentID       string            `json:"agent_id"`
	DownloadToken string            `json:"download_token"`
	Endpoints     map[string]string `json:"endpoints"`
}

// CertificateResponse represents the certificate data sent to the agent
type CertificateResponse struct {
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
}

// RegistrationHandler handles agent registration and certificate distribution
type RegistrationHandler struct {
	agentService *services.AgentService
	caManager    *auth.CAManager
}

// NewRegistrationHandler creates a new instance of RegistrationHandler
func NewRegistrationHandler(agentService *services.AgentService, caManager *auth.CAManager) *RegistrationHandler {
	return &RegistrationHandler{
		agentService: agentService,
		caManager:    caManager,
	}
}

// HandleRegistration processes agent registration requests
func (h *RegistrationHandler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Failed to decode registration request: %v", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate claim code
	if err := h.agentService.ValidateClaimCode(ctx, req.ClaimCode); err != nil {
		debug.Error("Invalid claim code: %v", err)
		http.Error(w, "Invalid claim code", http.StatusUnauthorized)
		return
	}

	// Register agent
	agent, err := h.agentService.RegisterAgent(ctx, req.ClaimCode, req.Hostname)
	if err != nil {
		debug.Error("Failed to register agent: %v", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	// Generate download token
	downloadToken, err := h.agentService.CreateDownloadToken(ctx, agent.ID)
	if err != nil {
		debug.Error("Failed to generate download token: %v", err)
		http.Error(w, "Failed to generate download token", http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := RegistrationResponse{
		AgentID:       agent.ID,
		DownloadToken: downloadToken,
		Endpoints: map[string]string{
			"cert": "/api/agent/cert",
		},
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		debug.Error("Failed to encode registration response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	debug.Info("Agent registered successfully: %s", agent.ID)
}

// HandleCertificateDownload processes certificate download requests
func (h *RegistrationHandler) HandleCertificateDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get download token from header
	downloadToken := r.Header.Get("X-Download-Token")
	if downloadToken == "" {
		debug.Error("No download token provided")
		http.Error(w, "Download token required", http.StatusUnauthorized)
		return
	}

	// Validate download token and get agent ID
	agentID, err := h.agentService.ValidateDownloadToken(ctx, downloadToken)
	if err != nil {
		debug.Error("Invalid download token: %v", err)
		http.Error(w, "Invalid download token", http.StatusUnauthorized)
		return
	}

	// Generate agent certificate
	cert, key, err := h.caManager.GenerateAgentCertificate(ctx, agentID)
	if err != nil {
		debug.Error("Failed to generate agent certificate: %v", err)
		http.Error(w, "Failed to generate certificate", http.StatusInternalServerError)
		return
	}

	// Get CA certificate
	caCert, err := h.caManager.GetCACertificate()
	if err != nil {
		debug.Error("Failed to get CA certificate: %v", err)
		http.Error(w, "Failed to get CA certificate", http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := CertificateResponse{
		Certificate:   string(cert),
		PrivateKey:    string(key),
		CACertificate: string(caCert),
	}

	// Invalidate download token
	if err := h.agentService.InvalidateDownloadToken(ctx, downloadToken); err != nil {
		debug.Error("Failed to invalidate download token: %v", err)
		// Continue with response since the certificate generation was successful
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		debug.Error("Failed to encode certificate response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	debug.Info("Certificates sent successfully to agent: %s", agentID)
}
