package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/websocket"
)

// Default connection timing values
const (
	defaultWriteWait  = 10 * time.Second
	defaultPongWait   = 60 * time.Second
	defaultPingPeriod = 54 * time.Second
	maxMessageSize    = 512 * 1024 // 512KB
)

// Connection timing configuration
var (
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration
)

// getEnvDuration gets a duration from an environment variable with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	debug.Info("Attempting to load environment variable: %s", key)
	value := os.Getenv(key)
	debug.Info("Environment variable %s value: %q", key, value)

	if value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			debug.Info("Successfully parsed %s: %v", key, duration)
			return duration
		}
		debug.Warning("Invalid %s value: %s, using default: %v", key, value, defaultValue)
	}
	debug.Info("No %s environment variable found, using default: %v", key, defaultValue)
	return defaultValue
}

// initTimingConfig initializes the timing configuration from environment variables
func initTimingConfig() {
	debug.Info("Initializing WebSocket timing configuration")
	writeWait = getEnvDuration("KH_WRITE_WAIT", defaultWriteWait)
	pongWait = getEnvDuration("KH_PONG_WAIT", defaultPongWait)
	pingPeriod = getEnvDuration("KH_PING_PERIOD", defaultPingPeriod)
	debug.Info("WebSocket timing configuration initialized:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
	HandshakeTimeout: writeWait,
	// TLS configuration is handled by the server
}

// Handler manages WebSocket connections for agents
type Handler struct {
	wsService    *wsservice.Service
	agentService *services.AgentService
	tlsConfig    *tls.Config
	clients      map[int]*Client
	mu           sync.RWMutex
}

// Client represents a connected agent
type Client struct {
	handler *Handler
	conn    *websocket.Conn
	agent   *models.Agent
	send    chan *wsservice.Message
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewHandler creates a new WebSocket handler
func NewHandler(wsService *wsservice.Service, agentService *services.AgentService, tlsConfig *tls.Config) *Handler {
	// Initialize timing configuration
	initTimingConfig()

	return &Handler{
		wsService:    wsService,
		agentService: agentService,
		tlsConfig:    tlsConfig,
		clients:      make(map[int]*Client),
	}
}

// ServeWS handles WebSocket connections from agents
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	debug.Info("Starting WebSocket connection with timing configuration:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)

	debug.Info("New WebSocket connection attempt received from %s", r.RemoteAddr)
	debug.Debug("Request headers: %v", r.Header)

	if h.tlsConfig != nil {
		if r.TLS == nil {
			debug.Error("TLS connection required but not provided from %s", r.RemoteAddr)
			http.Error(w, "TLS required", http.StatusBadRequest)
			return
		}
		debug.Info("TLS connection verified for %s", r.RemoteAddr)
	}

	// Get API key from header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		debug.Error("No API key provided from %s", r.RemoteAddr)
		http.Error(w, "API key required", http.StatusUnauthorized)
		return
	}

	// Get agent ID from header
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		debug.Error("No agent ID provided from %s", r.RemoteAddr)
		http.Error(w, "Agent ID required", http.StatusUnauthorized)
		return
	}

	// Validate API key and get agent
	agent, err := h.agentService.GetByAPIKey(r.Context(), apiKey)
	if err != nil {
		debug.Error("Invalid API key from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Verify agent ID matches
	if fmt.Sprintf("%d", agent.ID) != agentID {
		debug.Error("Agent ID mismatch from %s: provided=%s, actual=%d", r.RemoteAddr, agentID, agent.ID)
		http.Error(w, "Invalid agent ID", http.StatusUnauthorized)
		return
	}

	debug.Info("API key validated for agent %d from %s", agent.ID, r.RemoteAddr)

	// Configure WebSocket upgrader
	upgrader.EnableCompression = true

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		debug.Error("Failed to upgrade connection from %s: %v", r.RemoteAddr, err)
		return
	}
	debug.Info("Successfully upgraded to WebSocket connection for agent %d", agent.ID)

	// Create client context
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		handler: h,
		conn:    conn,
		agent:   agent,
		send:    make(chan *wsservice.Message, 256),
		ctx:     ctx,
		cancel:  cancel,
	}

	// If this is the agent's first connection and it has a claim code in metadata, mark it as used
	if agent.Status == models.AgentStatusPending {
		debug.Info("Processing first-time connection for agent %d", agent.ID)
		if claimCode, ok := agent.Metadata["claim_code"]; ok {
			debug.Info("Found claim code for agent %d", agent.ID)
			debug.Debug("Claim code details: %s", claimCode)
			if err := h.agentService.MarkClaimCodeUsed(ctx, claimCode, agent.ID); err != nil {
				debug.Error("Failed to mark claim code as used for agent %d: %v", agent.ID, err)
			} else {
				debug.Info("Successfully marked claim code as used for agent %d", agent.ID)
			}
			// Remove claim code from metadata as it's no longer needed
			delete(agent.Metadata, "claim_code")
			if err := h.agentService.Update(ctx, agent); err != nil {
				debug.Error("Failed to update agent metadata for agent %d: %v", agent.ID, err)
			} else {
				debug.Info("Successfully updated agent %d status", agent.ID)
			}
		}
	}

	debug.Info("Agent %d fully registered and ready", agent.ID)

	// Update agent status to active and update heartbeat
	if err := h.agentService.UpdateAgentStatus(ctx, agent.ID, models.AgentStatusActive, nil); err != nil {
		debug.Error("Failed to update agent status to active: %v", err)
	} else {
		debug.Info("Successfully updated agent %d status to active", agent.ID)
	}
	
	// Update heartbeat timestamp
	if err := h.agentService.UpdateHeartbeat(ctx, agent.ID); err != nil {
		debug.Error("Failed to update agent heartbeat: %v", err)
	} else {
		debug.Info("Successfully updated heartbeat for agent %d", agent.ID)
	}

	// Register client
	h.mu.Lock()
	h.clients[agent.ID] = client
	h.mu.Unlock()
	debug.Info("Added agent %d to active clients", agent.ID)

	// Start client routines
	go client.writePump()
	go client.readPump()

	// Initiate file sync with agent
	go h.initiateFileSync(client)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		debug.Info("Agent %d: ReadPump closing", c.agent.ID)
		// Update agent status to inactive when connection is closed
		if err := c.handler.agentService.UpdateAgentStatus(c.ctx, c.agent.ID, models.AgentStatusInactive, nil); err != nil {
			debug.Error("Failed to update agent status to inactive: %v", err)
		} else {
			debug.Info("Successfully updated agent %d status to inactive", c.agent.ID)
		}
		c.handler.unregisterClient(c)
		c.conn.Close()
		c.cancel()
	}()

	debug.Info("Agent %d: Starting readPump with timing configuration:", c.agent.ID)
	debug.Info("Agent %d: - Write Wait: %v", c.agent.ID, writeWait)
	debug.Info("Agent %d: - Pong Wait: %v", c.agent.ID, pongWait)
	debug.Info("Agent %d: - Ping Period: %v", c.agent.ID, pingPeriod)

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Set up ping handler
	c.conn.SetPingHandler(func(appData string) error {
		debug.Info("Agent %d: Received ping from client, sending pong", c.agent.ID)
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			debug.Error("Agent %d: Failed to set read deadline: %v", c.agent.ID, err)
			return err
		}
		// Send pong response immediately
		err = c.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(writeWait))
		if err != nil {
			debug.Error("Agent %d: Failed to send pong: %v", c.agent.ID, err)
			return err
		}
		debug.Info("Agent %d: Successfully sent pong response", c.agent.ID)
		return nil
	})

	// Set up pong handler
	c.conn.SetPongHandler(func(string) error {
		debug.Info("Agent %d: Received pong", c.agent.ID)
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			debug.Error("Agent %d: Failed to set read deadline: %v", c.agent.ID, err)
			return err
		}
		
		// Update heartbeat when we receive a pong (proof of active connection)
		if err := c.handler.agentService.UpdateHeartbeat(c.ctx, c.agent.ID); err != nil {
			debug.Error("Agent %d: Failed to update heartbeat on pong: %v", c.agent.ID, err)
			// Don't return error - pong handling should continue
		} else {
			debug.Info("Agent %d: Updated heartbeat on pong", c.agent.ID)
		}
		
		debug.Info("Agent %d: Successfully updated read deadline after pong", c.agent.ID)
		return nil
	})

	debug.Info("Agent %d: Ping/Pong handlers configured", c.agent.ID)

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				debug.Error("Agent %d: Unexpected WebSocket close error: %v", c.agent.ID, err)
			} else {
				debug.Info("Agent %d: Connection closed: %v", c.agent.ID, err)
			}
			break
		}
		debug.Debug("Agent %d: Received message type %d: %s", c.agent.ID, messageType, string(data))

		var msg wsservice.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			debug.Error("Agent %d: Failed to unmarshal message: %v", c.agent.ID, err)
			continue
		}

		debug.Info("Agent %d: Processing message type: %s", c.agent.ID, msg.Type)

		// Handle message based on type
		if err := c.handler.wsService.HandleMessage(c.ctx, c.agent, &msg); err != nil {
			debug.Error("Agent %d: Failed to handle message: %v", c.agent.ID, err)
		} else {
			debug.Info("Agent %d: Successfully processed message type: %s", c.agent.ID, msg.Type)
		}

		// Handle different message types
		switch msg.Type {
		case wsservice.TypeSyncResponse:
			c.handler.handleSyncResponse(c, &msg)

		case wsservice.TypeSyncStatus:
			c.handler.handleSyncStatus(c, &msg)

		default:
			// Handle other message types
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		debug.Info("Agent %d: WritePump closing", c.agent.ID)
		ticker.Stop()
		c.conn.Close()
	}()

	debug.Info("Agent %d: Starting writePump with timing configuration:", c.agent.ID)
	debug.Info("Agent %d: - Write Wait: %v", c.agent.ID, writeWait)
	debug.Info("Agent %d: - Pong Wait: %v", c.agent.ID, pongWait)
	debug.Info("Agent %d: - Ping Period: %v", c.agent.ID, pingPeriod)

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				debug.Info("Agent %d: Send channel closed", c.agent.ID)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				debug.Error("Agent %d: Failed to get next writer: %v", c.agent.ID, err)
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				debug.Error("Agent %d: Failed to marshal message: %v", c.agent.ID, err)
				continue
			}

			debug.Info("Agent %d: Sending message type: %s", c.agent.ID, message.Type)
			debug.Debug("Message details - Length: %d bytes", len(data))

			if _, err := w.Write(data); err != nil {
				debug.Error("Agent %d: Failed to write message: %v", c.agent.ID, err)
				return
			}

			if err := w.Close(); err != nil {
				debug.Error("Agent %d: Failed to close writer: %v", c.agent.ID, err)
				return
			}

			debug.Info("Agent %d: Successfully sent message type: %s", c.agent.ID, message.Type)

		case <-ticker.C:
			debug.Info("Agent %d: Sending ping", c.agent.ID)
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				debug.Error("Agent %d: Failed to send ping: %v", c.agent.ID, err)
				return
			}
			debug.Info("Agent %d: Successfully sent ping", c.agent.ID)

		case <-c.ctx.Done():
			debug.Info("Agent %d: Context cancelled", c.agent.ID)
			return
		}
	}
}

// SendMessage sends a message to a specific agent
func (h *Handler) SendMessage(agentID int, msg *wsservice.Message) error {
	h.mu.RLock()
	client, ok := h.clients[agentID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %d not connected", agentID)
	}

	select {
	case client.send <- msg:
		return nil
	default:
		return fmt.Errorf("agent %d send buffer full", agentID)
	}
}

// Broadcast sends a message to all connected agents
func (h *Handler) Broadcast(msg *wsservice.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- msg:
		default:
			debug.Error("failed to broadcast to agent %d: send buffer full", client.agent.ID)
		}
	}
}

// unregisterClient removes a client from the handler
func (h *Handler) unregisterClient(c *Client) {
	h.mu.Lock()
	if client, ok := h.clients[c.agent.ID]; ok {
		if client == c {
			delete(h.clients, c.agent.ID)
		}
	}
	h.mu.Unlock()
}

// GetConnectedAgents returns a list of connected agent IDs
func (h *Handler) GetConnectedAgents() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]int, 0, len(h.clients))
	for agentID := range h.clients {
		agents = append(agents, agentID)
	}
	return agents
}

// initiateFileSync starts the file synchronization process with an agent
func (h *Handler) initiateFileSync(client *Client) {
	debug.Info("Initiating file sync with agent %d", client.agent.ID)

	// Create a unique request ID
	requestID := fmt.Sprintf("sync-%d-%d", client.agent.ID, time.Now().UnixNano())

	// Create sync request payload
	payload := wsservice.FileSyncRequestPayload{
		RequestID: requestID,
		FileTypes: []string{"wordlist", "rule", "binary"},
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		debug.Error("Failed to marshal file sync request payload: %v", err)
		return
	}

	// Create message
	msg := &wsservice.Message{
		Type:    wsservice.TypeSyncRequest,
		Payload: payloadBytes,
	}

	// Send message to agent
	select {
	case client.send <- msg:
		debug.Info("Sent file sync request to agent %d", client.agent.ID)
	case <-client.ctx.Done():
		debug.Warning("Failed to send file sync request: agent %d disconnected", client.agent.ID)
	}
}

// handleSyncResponse processes a file sync response from an agent
func (h *Handler) handleSyncResponse(client *Client, msg *wsservice.Message) {
	var payload wsservice.FileSyncResponsePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		debug.Error("Failed to unmarshal file sync response: %v", err)
		return
	}

	debug.Info("Received file sync response from agent %d: %d files", client.agent.ID, len(payload.Files))

	// Determine which files need to be synced
	filesToSync, err := h.determineFilesToSync(client.agent.ID, payload.Files)
	if err != nil {
		debug.Error("Failed to determine files to sync: %v", err)
		return
	}

	if len(filesToSync) == 0 {
		debug.Info("Agent %d is up to date, no files to sync", client.agent.ID)
		return
	}

	// Create sync command payload
	commandPayload := wsservice.FileSyncCommandPayload{
		RequestID: fmt.Sprintf("sync-cmd-%d-%d", client.agent.ID, time.Now().UnixNano()),
		Action:    "download",
		Files:     filesToSync,
	}

	// Marshal payload
	commandBytes, err := json.Marshal(commandPayload)
	if err != nil {
		debug.Error("Failed to marshal file sync command payload: %v", err)
		return
	}

	// Create message
	command := &wsservice.Message{
		Type:    wsservice.TypeSyncCommand,
		Payload: commandBytes,
	}

	// Send message to agent
	select {
	case client.send <- command:
		debug.Info("Sent file sync command to agent %d to download %d files", client.agent.ID, len(filesToSync))
	case <-client.ctx.Done():
		debug.Warning("Failed to send file sync command: agent %d disconnected", client.agent.ID)
	}
}

// handleSyncStatus processes a file sync status update from an agent
func (h *Handler) handleSyncStatus(client *Client, msg *wsservice.Message) {
	var payload wsservice.FileSyncStatusPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		debug.Error("Failed to unmarshal file sync status: %v", err)
		return
	}

	debug.Info("File sync status update from agent %d: %s (%d%%)",
		client.agent.ID, payload.Status, payload.Progress)

	// If sync is complete, update agent status
	if payload.Status == "completed" {
		debug.Info("File sync completed for agent %d", client.agent.ID)
		// TODO: Update agent sync status in database
	}
}

// determineFilesToSync compares agent files with the backend and returns files that need syncing
func (h *Handler) determineFilesToSync(agentID int, agentFiles []wsservice.FileInfo) ([]wsservice.FileInfo, error) {
	// Get files from backend
	backendFiles, err := h.getBackendFiles(context.Background(), []string{"wordlist", "rule", "binary"}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get backend files: %w", err)
	}

	// Create a map of agent files for quick lookup
	agentFileMap := make(map[string]wsservice.FileInfo)
	for _, file := range agentFiles {
		key := fmt.Sprintf("%s:%s", file.FileType, file.Name)
		agentFileMap[key] = file
	}

	// Determine which files need to be synced
	var filesToSync []wsservice.FileInfo

	for _, file := range backendFiles {
		key := fmt.Sprintf("%s:%s", file.FileType, file.Name)
		agentFile, exists := agentFileMap[key]

		// If the file doesn't exist on agent or MD5 hash doesn't match, add to sync list
		if !exists || agentFile.MD5Hash != file.MD5Hash {
			filesToSync = append(filesToSync, file)
		}
	}

	return filesToSync, nil
}

// getBackendFiles retrieves files from the backend database based on file types
func (h *Handler) getBackendFiles(ctx context.Context, fileTypes []string, category string) ([]wsservice.FileInfo, error) {
	debug.Info("Retrieving backend files for types: %v, category: %s", fileTypes, category)

	// Retrieve files using the agent service
	repoFiles, err := h.agentService.GetFiles(ctx, fileTypes, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get files from repository: %w", err)
	}

	// Convert repository.FileInfo to wsservice.FileInfo
	var files []wsservice.FileInfo
	for _, file := range repoFiles {
		files = append(files, wsservice.FileInfo{
			Name:      file.Name,
			MD5Hash:   file.MD5Hash,
			Size:      file.Size,
			FileType:  file.FileType,
			Category:  file.Category,
			ID:        file.ID,
			Timestamp: file.Timestamp,
		})
	}

	debug.Info("Retrieved %d files from repository", len(files))
	return files, nil
}
