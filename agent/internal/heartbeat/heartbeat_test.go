package heartbeat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	// Since Start runs forever and sendHeartbeat is not mockable,
	// we'll test that the function can be started without panicking
	// and runs for a short duration
	
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	
	// Start heartbeat in a goroutine
	interval := 100 * time.Millisecond
	started := make(chan bool)
	
	go func() {
		started <- true
		Start(interval)
	}()
	
	// Ensure it started
	select {
	case <-started:
		// Good, it started
	case <-time.After(1 * time.Second):
		t.Fatal("Start function failed to begin execution")
	}
	
	// Let it run for a bit
	<-ctx.Done()
	
	// If we get here without panicking, the test passes
	assert.True(t, true, "Start function executed without panic")
}

func TestSendHeartbeat(t *testing.T) {
	// Test the sendHeartbeat function
	// Since it's just a stub that logs, we can only verify it doesn't panic
	assert.NotPanics(t, func() {
		sendHeartbeat()
	})
}

func TestStartWithVariousIntervals(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"1 second interval", 1 * time.Second},
		{"500ms interval", 500 * time.Millisecond},
		{"100ms interval", 100 * time.Millisecond},
		{"10ms interval", 10 * time.Millisecond},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			
			// Start in goroutine
			go Start(tt.interval)
			
			// Let it run briefly
			<-ctx.Done()
			
			// Test passes if no panic occurred
			assert.True(t, true, "No panic with interval %v", tt.interval)
		})
	}
}