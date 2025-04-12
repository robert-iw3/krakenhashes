package status

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/websocket"
)

// StatusHandler manages real-time status updates
type StatusHandler struct {
	mu            sync.RWMutex
	clients       map[*websocket.Conn]bool
	hashlistRepo  repository.HashListRepository
	notifyChan    chan Notification
	batchInterval time.Duration
	batchSize     int
}

// Notification represents a status update notification
type Notification struct {
	HashlistID int64
	Message    string
	Data       interface{}
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(
	hashlistRepo repository.HashListRepository,
	batchInterval time.Duration,
	batchSize int,
) *StatusHandler {
	return &StatusHandler{
		clients:       make(map[*websocket.Conn]bool),
		hashlistRepo:  hashlistRepo,
		notifyChan:    make(chan Notification, 100),
		batchInterval: batchInterval,
		batchSize:     batchSize,
	}
}

// HandleWebSocket manages WebSocket connections
func (h *StatusHandler) HandleWebSocket(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			debug.Error("WebSocket read error: %v", err)
			break
		}

		var update models.StatusUpdate
		if err := json.Unmarshal(msg, &update); err != nil {
			debug.Error("Invalid status update format: %v", err)
			continue
		}

		if err := h.processUpdate(context.Background(), update); err != nil {
			debug.Error("Failed to process update: %v", err)
		}
	}
}

// processUpdate handles a single status update
func (h *StatusHandler) processUpdate(ctx context.Context, update models.StatusUpdate) error {
	// Validate update
	if update.HashlistID <= 0 {
		return fmt.Errorf("invalid hashlist ID: %d", update.HashlistID)
	}

	// Send to batch processor
	h.notifyChan <- Notification{
		HashlistID: update.HashlistID,
		Message:    "status_update",
		Data:       update,
	}

	return nil
}

// StartBatchProcessor runs the batch update routine
func (h *StatusHandler) StartBatchProcessor(ctx context.Context) {
	ticker := time.NewTicker(h.batchInterval)
	defer ticker.Stop()

	var batch []Notification

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(batch) > 0 {
				h.processBatch(ctx, batch)
				batch = batch[:0]
			}
		case notification := <-h.notifyChan:
			batch = append(batch, notification)
			if len(batch) >= h.batchSize {
				h.processBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// processBatch executes bulk database updates
func (h *StatusHandler) processBatch(ctx context.Context, batch []Notification) {
	updates := make(map[int64]*models.StatusUpdate)

	// Deduplicate and merge updates
	for _, n := range batch {
		updateData, ok := n.Data.(models.StatusUpdate)
		if !ok {
			debug.Warning("Invalid data type in notification batch: expected StatusUpdate, got %T", n.Data)
			continue
		}

		if existing, ok := updates[n.HashlistID]; ok {
			// Merge updates
			existing.CrackedCount += updateData.CrackedCount
			if updateData.Status != "" {
				existing.Status = updateData.Status
			}
		} else {
			newUpdate := updateData
			updates[n.HashlistID] = &newUpdate
		}
	}

	// Bulk update database
	for hashlistID, update := range updates {
		// Increment cracked count if new cracks were reported in this batch
		if update.CrackedCount > 0 {
			if err := h.hashlistRepo.IncrementCrackedCount(ctx, hashlistID, update.CrackedCount); err != nil {
				debug.Error("Batch increment cracked count failed for %d: %v", hashlistID, err)
			}
		}
		// Update status if a new status was provided
		if update.Status != "" {
			if err := h.hashlistRepo.UpdateStatus(ctx, hashlistID, update.Status, ""); err != nil {
				debug.Error("Batch status update failed for %d: %v", hashlistID, err)
			}
		}
	}

	// Broadcast notifications
	h.broadcastUpdates(batch)
}

// broadcastUpdates sends notifications to connected clients
func (h *StatusHandler) broadcastUpdates(notifications []Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, n := range notifications {
		msg, err := json.Marshal(n)
		if err != nil {
			debug.Error("Failed to marshal notification: %v", err)
			continue
		}

		for client := range h.clients {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				debug.Error("WebSocket write error: %v", err)
			}
		}
	}
}
