package agent

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

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

// GetAgent retrieves an agent by ID
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse agent ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Get agent
	agent, err := h.service.GetAgent(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// DeleteAgent deletes an agent
func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse agent ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Delete agent
	if err := h.service.DeleteAgent(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
