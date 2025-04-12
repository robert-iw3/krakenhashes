package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// CrackUpdateMessage defines the structure agents send for cracked hashes.
type CrackUpdateMessage struct {
	HashlistID int64  `json:"hashlist_id"` // ID of the hashlist this hash belongs to (Changed to int64)
	HashValue  string `json:"hash_value"`  // The actual hash value that was cracked
	Password   string `json:"password"`    // The cracked password
}

type AgentUpdateHandler struct {
	db           *db.DB
	agentService *services.AgentService
	hashRepo     *repository.HashRepository
	hashlistRepo *repository.HashListRepository
	upgrader     websocket.Upgrader
}

func NewAgentUpdateHandler(database *db.DB, agentService *services.AgentService, hashRepo *repository.HashRepository, hashlistRepo *repository.HashListRepository) *AgentUpdateHandler {
	// Configure the upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// CheckOrigin should be implemented for security in production
		// It verifies the origin of the WebSocket request.
		CheckOrigin: func(r *http.Request) bool {
			// Allow all connections for now, but restrict in production
			// origin := r.Header.Get("Origin")
			// debug.Debug("Checking WebSocket origin: %s", origin)
			// return origin == "expected-agent-origin-or-empty"
			return true
		},
	}
	return &AgentUpdateHandler{
		db:           database,
		agentService: agentService,
		hashRepo:     hashRepo,
		hashlistRepo: hashlistRepo,
		upgrader:     upgrader,
	}
}

// RegisterAgentUpdateRoutes sets up the WebSocket endpoint for agent updates.
func RegisterAgentUpdateRoutes(r *mux.Router, h *AgentUpdateHandler) {
	debug.Info("Registering WebSocket route for agent crack updates")
	// Note: Authentication needs to be applied here, similar to REST routes
	// We might need a WebSocket-specific middleware adapter.
	r.HandleFunc("/ws/agent/updates", h.HandleUpdates)
}

// HandleUpdates handles incoming WebSocket connections from agents sending updates.
func (h *AgentUpdateHandler) HandleUpdates(w http.ResponseWriter, r *http.Request) {
	// --- Agent Authentication for WebSocket (before upgrading) ---
	apiKey := r.Header.Get("X-API-Key")
	agentIDStr := r.Header.Get("X-Agent-ID")

	if apiKey == "" || agentIDStr == "" {
		debug.Warning("WebSocket Upgrade: Agent request missing X-API-Key or X-Agent-ID header")
		http.Error(w, "API Key and Agent ID required", http.StatusUnauthorized)
		return
	}

	agentIDHeader, err := strconv.Atoi(agentIDStr)
	if err != nil {
		debug.Warning("WebSocket Upgrade: Agent request with invalid X-Agent-ID format: %s", agentIDStr)
		http.Error(w, "Invalid Agent ID format", http.StatusUnauthorized)
		return
	}

	agent, err := h.agentService.GetByAPIKey(r.Context(), apiKey)
	if err != nil {
		debug.Warning("WebSocket Upgrade: Invalid API Key provided (agent ID %d): %v", agentIDHeader, err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		} else {
			debug.Error("WebSocket Upgrade: Error validating agent API key: %v", err)
			http.Error(w, "Error validating agent credentials", http.StatusInternalServerError)
		}
		return
	}
	if agent == nil { // Should not happen if GetByAPIKey is correct
		debug.Error("WebSocket Upgrade: Agent not found for valid API key (agent ID %d)", agentIDHeader)
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	if agent.ID != agentIDHeader {
		debug.Warning("WebSocket Upgrade: Agent ID mismatch: Header (%d) != API Key Owner (%d)", agentIDHeader, agent.ID)
		http.Error(w, "API Key does not match Agent ID", http.StatusUnauthorized)
		return
	}
	// Authentication successful
	authenticatedAgentID := agent.ID
	debug.Info("Agent %d attempting WebSocket connection for updates", authenticatedAgentID)
	// --- End Authentication ---

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		debug.Error("Failed to upgrade WebSocket connection for agent %d: %v", authenticatedAgentID, err)
		// Don't write http error after hijack
		return
	}
	defer conn.Close()
	debug.Info("Agent %d WebSocket connection established for updates", authenticatedAgentID)

	// Configure connection limits (pong handling, message size)
	conn.SetReadLimit(maxMessageSize)              // Use a reasonable limit
	conn.SetReadDeadline(time.Now().Add(pongWait)) // Requires pong messages
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// TODO: Implement Ping ticker if needed

	// Read messages from the agent
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				debug.Error("Agent %d WebSocket read error: %v", authenticatedAgentID, err)
			} else {
				debug.Info("Agent %d WebSocket connection closed: %v", authenticatedAgentID, err)
			}
			break // Exit loop on error or close
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			var updateMsg CrackUpdateMessage
			if err := json.Unmarshal(message, &updateMsg); err != nil {
				debug.Warning("Agent %d: Failed to unmarshal crack update message: %v. Message: %s", authenticatedAgentID, err, string(message))
				continue
			}

			// Process the crack update
			debug.Debug("Received crack update from agent %d: %+v", authenticatedAgentID, updateMsg)
			if err := h.processCrackUpdate(context.Background(), updateMsg); err != nil {
				debug.Error("Agent %d: Failed to process crack update for hashlist %d, hash %s: %v", authenticatedAgentID, updateMsg.HashlistID, updateMsg.HashValue, err)
			}
		}
	}
	debug.Info("Agent %d WebSocket handler finished", authenticatedAgentID)
}

// processCrackUpdate handles the logic for updating the database based on a crack message.
func (h *AgentUpdateHandler) processCrackUpdate(ctx context.Context, msg CrackUpdateMessage) error {
	// Basic validation
	if msg.HashlistID <= 0 || msg.HashValue == "" || msg.Password == "" { // Check for positive ID
		return fmt.Errorf("invalid crack update message: missing required fields or invalid hashlist ID")
	}

	debug.Info("Processing crack update: Hashlist=%d, Hash=%s, Password=***", msg.HashlistID, msg.HashValue)

	// Reinstate transaction management
	var err error // Declare err here to be accessible in defer
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		debug.Error("Failed to begin transaction for crack update: %v", err)
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	// Defer rollback in case of errors
	defer func() {
		if err != nil { // Check if an error occurred within the function scope
			if rbErr := tx.Rollback(); rbErr != nil {
				debug.Error("Failed to rollback transaction after error: %v (original error: %v)", rbErr, err)
			}
		}
	}()

	// 1. Find the Hash entity by HashValue using the transaction.
	hash, err := h.hashRepo.GetByHashValueForUpdate(tx, msg.HashValue) // Use Tx variant
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			debug.Warning("Crack update ignored: Hash value '%s' not found.", msg.HashValue)
			// Set err to nil so Rollback isn't triggered mistakenly
			err = nil
			return nil
		} else {
			debug.Error("Failed to query hash '%s': %v", msg.HashValue, err)
			// err is already set
			return fmt.Errorf("failed to query hash: %w", err)
		}
	}

	// 2. If found and not already cracked:
	if hash.IsCracked {
		debug.Info("Crack update ignored: Hash value '%s' (ID: %s) is already marked as cracked.", msg.HashValue, hash.ID)
		// Set err to nil so Rollback isn't triggered mistakenly
		err = nil
		return nil
	}

	// 3a. Update the Hash using the transaction
	now := time.Now()
	plainTextPtr := &msg.Password
	err = h.hashRepo.UpdateCrackStatus(tx, hash.ID, msg.Password, now, plainTextPtr) // Pass tx
	if err != nil {
		debug.Error("Failed to update crack status for hash %s (ID: %s): %v", msg.HashValue, hash.ID, err)
		// err is already set
		return fmt.Errorf("failed to update hash status: %w", err)
	}
	debug.Debug("Updated crack status for hash %s (ID: %s)", msg.HashValue, hash.ID)

	// 3b. Increment the HashList CrackedHashes count using a transaction variant
	// NOTE: IncrementCrackedCountTx is a new method required in HashListRepository
	err = h.hashlistRepo.IncrementCrackedCountTx(tx, msg.HashlistID, 1) // Call Tx variant
	if err != nil {
		debug.Error("Failed to increment cracked count for hashlist %d after cracking hash %s: %v", msg.HashlistID, hash.ID, err)
		// err is already set, transaction will roll back.
		return fmt.Errorf("failed to increment hashlist count: %w", err)
	}
	debug.Debug("Incremented cracked count for hashlist %d", msg.HashlistID)

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		debug.Error("Failed to commit transaction for crack update (Hashlist: %d, Hash: %s): %v", msg.HashlistID, hash.ID, err)
		// err is already set
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	debug.Info("Successfully processed crack update: Hashlist=%d, Hash=%s", msg.HashlistID, msg.HashValue)
	return nil // Success
}

// Define constants for WebSocket timings and limits (consider moving to config)
const (
	maxMessageSize = 8192                // Max message size in bytes
	pongWait       = 60 * time.Second    // Time allowed to read the next pong message
	pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
)
