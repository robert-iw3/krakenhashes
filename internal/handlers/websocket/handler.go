package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	wsservice "github.com/ZerkerEOD/hashdom-backend/internal/services/websocket"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Handler manages WebSocket connections for agents
type Handler struct {
	wsService *wsservice.Service
	clients   map[string]*Client
	mu        sync.RWMutex
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
func NewHandler(wsService *wsservice.Service) *Handler {
	return &Handler{
		wsService: wsService,
		clients:   make(map[string]*Client),
	}
}

// ServeWS handles WebSocket connections from agents
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Get agent from context (set by cert auth middleware)
	agent, ok := r.Context().Value("agent").(*models.Agent)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		debug.Error("failed to upgrade connection: %v", err)
		return
	}

	// Create client context
	ctx, cancel := context.WithCancel(r.Context())

	client := &Client{
		handler: h,
		conn:    conn,
		agent:   agent,
		send:    make(chan *wsservice.Message, 256),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Register client
	h.mu.Lock()
	h.clients[agent.ID] = client
	h.mu.Unlock()

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.handler.unregisterClient(c)
		c.conn.Close()
		c.cancel()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				debug.Error("unexpected close error: %v", err)
			}
			return
		}

		var msg wsservice.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			debug.Error("failed to unmarshal message: %v", err)
			continue
		}

		// Handle message based on type
		if err := c.handler.wsService.HandleMessage(c.ctx, c.agent, &msg); err != nil {
			debug.Error("failed to handle message: %v", err)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				debug.Error("failed to marshal message: %v", err)
				continue
			}

			if _, err := w.Write(data); err != nil {
				return
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// SendMessage sends a message to a specific agent
func (h *Handler) SendMessage(agentID string, msg *wsservice.Message) error {
	h.mu.RLock()
	client, ok := h.clients[agentID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	select {
	case client.send <- msg:
		return nil
	default:
		return fmt.Errorf("agent %s send buffer full", agentID)
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
func (h *Handler) GetConnectedAgents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]string, 0, len(h.clients))
	for agentID := range h.clients {
		agents = append(agents, agentID)
	}
	return agents
}
