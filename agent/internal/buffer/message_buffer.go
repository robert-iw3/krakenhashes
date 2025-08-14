package buffer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
	"github.com/google/uuid"
)

// MessageType represents the type of buffered message
type MessageType string

const (
	// Critical message types that must be buffered
	MessageTypeJobProgress MessageType = "job_progress"
	MessageTypeHashcatOutput MessageType = "hashcat_output"
	MessageTypeBenchmarkResult MessageType = "benchmark_result"
)

// BufferedMessage represents a message stored in the buffer
type BufferedMessage struct {
	ID        string          `json:"id"`
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	AgentID   int             `json:"agent_id,omitempty"`
}

// MessageBuffer manages persistent storage of messages during disconnections
type MessageBuffer struct {
	mu       sync.RWMutex
	messages []BufferedMessage
	filePath string
	agentID  int
}

// NewMessageBuffer creates a new message buffer instance
func NewMessageBuffer(dataDir string, agentID int) (*MessageBuffer, error) {
	bufferPath := filepath.Join(dataDir, "message_buffer.json")
	
	mb := &MessageBuffer{
		filePath: bufferPath,
		agentID:  agentID,
		messages: make([]BufferedMessage, 0),
	}
	
	// Load existing buffer if it exists
	if err := mb.LoadFromDisk(); err != nil {
		debug.Warning("Failed to load existing buffer (will start fresh): %v", err)
	}
	
	return mb, nil
}

// Add adds a new message to the buffer and persists to disk
func (mb *MessageBuffer) Add(msgType MessageType, payload interface{}) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	msg := BufferedMessage{
		ID:        uuid.New().String(),
		Type:      msgType,
		Payload:   json.RawMessage(payloadBytes),
		Timestamp: time.Now().UTC(),
		AgentID:   mb.agentID,
	}
	
	mb.messages = append(mb.messages, msg)
	
	// Persist to disk
	if err := mb.saveToDiskLocked(); err != nil {
		// Remove the message if we can't persist it
		mb.messages = mb.messages[:len(mb.messages)-1]
		return fmt.Errorf("failed to persist buffer: %w", err)
	}
	
	debug.Info("Buffered message %s of type %s (total buffered: %d)", msg.ID, msgType, len(mb.messages))
	return nil
}

// GetAll returns all buffered messages
func (mb *MessageBuffer) GetAll() []BufferedMessage {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make([]BufferedMessage, len(mb.messages))
	copy(result, mb.messages)
	return result
}

// Clear removes all messages from the buffer and disk
func (mb *MessageBuffer) Clear() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	mb.messages = make([]BufferedMessage, 0)
	
	// Remove the file
	if err := os.Remove(mb.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove buffer file: %w", err)
	}
	
	debug.Info("Cleared message buffer")
	return nil
}

// RemoveMessages removes specific messages by their IDs
func (mb *MessageBuffer) RemoveMessages(ids []string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	
	newMessages := make([]BufferedMessage, 0, len(mb.messages))
	removed := 0
	
	for _, msg := range mb.messages {
		if !idMap[msg.ID] {
			newMessages = append(newMessages, msg)
		} else {
			removed++
		}
	}
	
	mb.messages = newMessages
	
	// Persist changes
	if err := mb.saveToDiskLocked(); err != nil {
		return fmt.Errorf("failed to persist buffer after removal: %w", err)
	}
	
	debug.Info("Removed %d messages from buffer (%d remaining)", removed, len(mb.messages))
	return nil
}

// LoadFromDisk loads buffered messages from disk
func (mb *MessageBuffer) LoadFromDisk() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	data, err := os.ReadFile(mb.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No buffer file exists, which is fine
			return nil
		}
		return fmt.Errorf("failed to read buffer file: %w", err)
	}
	
	if len(data) == 0 {
		// Empty file, nothing to load
		return nil
	}
	
	var messages []BufferedMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to unmarshal buffer: %w", err)
	}
	
	mb.messages = messages
	debug.Info("Loaded %d messages from buffer", len(messages))
	return nil
}

// SaveToDisk persists the current buffer to disk
func (mb *MessageBuffer) SaveToDisk() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return mb.saveToDiskLocked()
}

// saveToDiskLocked saves to disk (caller must hold lock)
func (mb *MessageBuffer) saveToDiskLocked() error {
	// If buffer is empty, remove the file
	if len(mb.messages) == 0 {
		if err := os.Remove(mb.filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove empty buffer file: %w", err)
		}
		return nil
	}
	
	data, err := json.MarshalIndent(mb.messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal buffer: %w", err)
	}
	
	// Write to temp file first for atomicity
	tempFile := mb.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp buffer file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, mb.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename buffer file: %w", err)
	}
	
	return nil
}

// Count returns the number of buffered messages
func (mb *MessageBuffer) Count() int {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return len(mb.messages)
}

// IsCriticalMessage determines if a message type should be buffered
func IsCriticalMessage(msgType string) bool {
	switch MessageType(msgType) {
	case MessageTypeJobProgress,
	     MessageTypeHashcatOutput,
	     MessageTypeBenchmarkResult:
		return true
	default:
		return false
	}
}

// HasCrackedHashes checks if a job progress message contains crack information
func HasCrackedHashes(payload json.RawMessage) bool {
	var progress struct {
		CrackedCount int      `json:"cracked_count"`
		CrackedHashes []string `json:"cracked_hashes"`
	}
	
	if err := json.Unmarshal(payload, &progress); err != nil {
		return false
	}
	
	return progress.CrackedCount > 0 || len(progress.CrackedHashes) > 0
}