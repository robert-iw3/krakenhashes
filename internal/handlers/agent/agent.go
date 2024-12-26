package agent

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/mux"
)

type AgentHandler struct {
	service *services.AgentService
}

func NewAgentHandler(service *services.AgentService) *AgentHandler {
	return &AgentHandler{service: service}
}

// ListAgents handles listing all agents
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	debug.Info("Listing agents")

	// Parse query parameters into filters
	filters := make(map[string]interface{})
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if team := r.URL.Query().Get("team"); team != "" {
		filters["team"] = team
	}

	agents, err := h.service.ListAgents(r.Context(), filters)
	if err != nil {
		debug.Error("failed to list agents: %v", err)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}

	// Set content type and encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		debug.Error("failed to encode agents: %v", err)
		http.Error(w, "Failed to encode agents", http.StatusInternalServerError)
		return
	}
}

// GetAgent handles retrieving a single agent
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		debug.Error("missing agent ID")
		http.Error(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	debug.Info("Getting agent: %s", id)

	agent, err := h.service.GetAgent(r.Context(), id)
	if err != nil {
		debug.Error("failed to get agent: %v", err)
		http.Error(w, "Failed to get agent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// DeleteAgent handles agent deletion
func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		debug.Error("missing agent ID")
		http.Error(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	debug.Info("Deleting agent: %s", id)

	if err := h.service.DeleteAgent(r.Context(), id); err != nil {
		debug.Error("failed to delete agent: %v", err)
		http.Error(w, "Failed to delete agent", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RegistrationRequest represents the agent registration request
type RegistrationRequest struct {
	ClaimCode string `json:"claim_code"`
	Hostname  string `json:"hostname"`
}

// RegistrationResponse represents the agent registration response
type RegistrationResponse struct {
	AgentID       string            `json:"agent_id"`
	DownloadToken string            `json:"download_token"`
	Endpoints     map[string]string `json:"endpoints"`
}

// RegisterAgent handles the initial registration of an agent using a claim code
func (h *AgentHandler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Failed to decode registration request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate claim code and register agent
	agent, err := h.service.RegisterAgent(r.Context(), req.ClaimCode, req.Hostname)
	if err != nil {
		debug.Error("Failed to register agent: %v", err)
		http.Error(w, "Registration failed", http.StatusUnauthorized)
		return
	}

	// Generate a temporary download token
	downloadToken, err := h.service.CreateDownloadToken(r.Context(), agent.ID)
	if err != nil {
		debug.Error("Failed to create download token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := RegistrationResponse{
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
	json.NewEncoder(w).Encode(response)
}
