package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

type AgentConnection struct {
	AgentID string
	Conn    *websocket.Conn
	mu      sync.Mutex
}

type AgentWebSocketHandler struct {
	agentService *services.AgentService
	connections  map[string]*AgentConnection
	mu           sync.RWMutex
}

func NewAgentWebSocketHandler(agentService *services.AgentService) *AgentWebSocketHandler {
	return &AgentWebSocketHandler{
		agentService: agentService,
		connections:  make(map[string]*AgentConnection),
	}
}

func (h *AgentWebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract agent ID from request
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		http.Error(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	// Verify agent exists and is active
	agent, err := h.agentService.GetAgent(r.Context(), agentID)
	if err != nil {
		debug.Error("Failed to get agent: %v", err)
		http.Error(w, "Invalid agent ID", http.StatusUnauthorized)
		return
	}

	if agent.Status != "active" {
		http.Error(w, "Agent is not active", http.StatusForbidden)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		debug.Error("Failed to upgrade connection: %v", err)
		return
	}

	// Create new agent connection
	agentConn := &AgentConnection{
		AgentID: agentID,
		Conn:    conn,
	}

	// Store connection
	h.mu.Lock()
	h.connections[agentID] = agentConn
	h.mu.Unlock()

	// Start heartbeat
	go h.handleHeartbeat(agentConn)

	// Handle messages
	go h.handleMessages(agentConn)
}

func (h *AgentWebSocketHandler) handleHeartbeat(ac *AgentConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ac.mu.Lock()
			err := ac.Conn.WriteJSON(map[string]string{"type": "ping"})
			ac.mu.Unlock()

			if err != nil {
				debug.Error("Failed to send heartbeat to agent %s: %v", ac.AgentID, err)
				h.removeConnection(ac.AgentID)
				return
			}
		}
	}
}

func (h *AgentWebSocketHandler) handleMessages(ac *AgentConnection) {
	defer h.removeConnection(ac.AgentID)

	for {
		messageType, message, err := ac.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				debug.Error("WebSocket error: %v", err)
			}
			return
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				debug.Error("Failed to parse message: %v", err)
				continue
			}

			// Handle different message types
			switch msg["type"] {
			case "pong":
				// Update last seen timestamp
				h.agentService.UpdateLastSeen(ac.AgentID)
			case "hardware_update":
				if hardware, ok := msg["hardware"].(map[string]interface{}); ok {
					h.agentService.UpdateHardwareInfo(ac.AgentID, hardware)
				}
			}
		}
	}
}

func (h *AgentWebSocketHandler) removeConnection(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[agentID]; exists {
		conn.Conn.Close()
		delete(h.connections, agentID)
	}
}

func (h *AgentWebSocketHandler) BroadcastMessage(message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conn := range h.connections {
		conn.mu.Lock()
		err := conn.Conn.WriteJSON(message)
		conn.mu.Unlock()

		if err != nil {
			debug.Error("Failed to broadcast to agent %s: %v", conn.AgentID, err)
		}
	}
}
