package buffer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMessageBuffer(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	agentID := 1
	
	// Create buffer
	mb, err := NewMessageBuffer(tempDir, agentID)
	if err != nil {
		t.Fatalf("Failed to create message buffer: %v", err)
	}
	
	// Test adding messages
	t.Run("AddMessages", func(t *testing.T) {
		payload1 := map[string]interface{}{
			"task_id": "test-task-1",
			"cracked_count": 2,
			"cracked_hashes": []string{"hash1:password1", "hash2:password2"},
		}
		
		err := mb.Add(MessageTypeJobProgress, payload1)
		if err != nil {
			t.Errorf("Failed to add message: %v", err)
		}
		
		if mb.Count() != 1 {
			t.Errorf("Expected 1 message, got %d", mb.Count())
		}
		
		// Add another message
		payload2 := map[string]interface{}{
			"task_id": "test-task-2",
			"status": "completed",
		}
		
		err = mb.Add(MessageTypeBenchmarkResult, payload2)
		if err != nil {
			t.Errorf("Failed to add second message: %v", err)
		}
		
		if mb.Count() != 2 {
			t.Errorf("Expected 2 messages, got %d", mb.Count())
		}
	})
	
	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		// Create new buffer instance to test loading
		mb2, err := NewMessageBuffer(tempDir, agentID)
		if err != nil {
			t.Fatalf("Failed to create second buffer: %v", err)
		}
		
		if mb2.Count() != 2 {
			t.Errorf("Expected 2 messages after loading, got %d", mb2.Count())
		}
		
		messages := mb2.GetAll()
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
		
		// Verify message content
		if messages[0].Type != MessageTypeJobProgress {
			t.Errorf("Expected first message type %s, got %s", MessageTypeJobProgress, messages[0].Type)
		}
	})
	
	// Test removing specific messages
	t.Run("RemoveMessages", func(t *testing.T) {
		messages := mb.GetAll()
		if len(messages) < 2 {
			t.Skip("Not enough messages to test removal")
		}
		
		// Remove first message
		idsToRemove := []string{messages[0].ID}
		err := mb.RemoveMessages(idsToRemove)
		if err != nil {
			t.Errorf("Failed to remove messages: %v", err)
		}
		
		if mb.Count() != 1 {
			t.Errorf("Expected 1 message after removal, got %d", mb.Count())
		}
		
		// Verify the right message was removed
		remaining := mb.GetAll()
		if remaining[0].ID == messages[0].ID {
			t.Errorf("Wrong message removed")
		}
	})
	
	// Test clearing
	t.Run("Clear", func(t *testing.T) {
		err := mb.Clear()
		if err != nil {
			t.Errorf("Failed to clear buffer: %v", err)
		}
		
		if mb.Count() != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", mb.Count())
		}
		
		// Verify file is removed
		bufferPath := filepath.Join(tempDir, "message_buffer.json")
		if _, err := os.Stat(bufferPath); !os.IsNotExist(err) {
			t.Errorf("Buffer file should be removed after clear")
		}
	})
	
	// Test critical message detection
	t.Run("CriticalMessageDetection", func(t *testing.T) {
		if !IsCriticalMessage("job_progress") {
			t.Errorf("job_progress should be critical")
		}
		
		if !IsCriticalMessage("hashcat_output") {
			t.Errorf("hashcat_output should be critical")
		}
		
		if IsCriticalMessage("heartbeat") {
			t.Errorf("heartbeat should not be critical")
		}
		
		if IsCriticalMessage("agent_status") {
			t.Errorf("agent_status should not be critical")
		}
	})
	
	// Test crack detection
	t.Run("CrackDetection", func(t *testing.T) {
		// Message with cracks
		withCracks := json.RawMessage(`{
			"task_id": "test",
			"cracked_count": 2,
			"cracked_hashes": ["hash1:pass1"]
		}`)
		
		if !HasCrackedHashes(withCracks) {
			t.Errorf("Should detect cracks in message")
		}
		
		// Message without cracks
		withoutCracks := json.RawMessage(`{
			"task_id": "test",
			"cracked_count": 0,
			"cracked_hashes": []
		}`)
		
		if HasCrackedHashes(withoutCracks) {
			t.Errorf("Should not detect cracks in message")
		}
	})
}

func TestBufferCorruption(t *testing.T) {
	tempDir := t.TempDir()
	bufferPath := filepath.Join(tempDir, "message_buffer.json")
	
	// Write corrupted JSON
	err := os.WriteFile(bufferPath, []byte(`{"invalid json`), 0600)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}
	
	// Try to load - should not crash
	mb, err := NewMessageBuffer(tempDir, 1)
	if err != nil {
		t.Fatalf("Should handle corrupted buffer gracefully: %v", err)
	}
	
	// Should start with empty buffer
	if mb.Count() != 0 {
		t.Errorf("Expected empty buffer after corruption, got %d messages", mb.Count())
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	mb, err := NewMessageBuffer(tempDir, 1)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	
	// Run concurrent operations
	done := make(chan bool, 3)
	
	// Writer 1
	go func() {
		for i := 0; i < 10; i++ {
			mb.Add(MessageTypeJobProgress, map[string]int{"id": i})
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()
	
	// Writer 2
	go func() {
		for i := 10; i < 20; i++ {
			mb.Add(MessageTypeHashcatOutput, map[string]int{"id": i})
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()
	
	// Reader
	go func() {
		for i := 0; i < 20; i++ {
			_ = mb.GetAll()
			_ = mb.Count()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
	
	// Verify final count
	if mb.Count() != 20 {
		t.Errorf("Expected 20 messages after concurrent access, got %d", mb.Count())
	}
}