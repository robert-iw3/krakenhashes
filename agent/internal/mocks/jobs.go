package mocks

import (
	"context"
	"sync"
)

// MockJobExecutor implements a mock job executor for testing
type MockJobExecutor struct {
	mu sync.RWMutex
	
	// Control behavior
	ExecuteFunc      func(ctx context.Context, task interface{}) error
	StopFunc         func() error
	GetStatusFunc    func() interface{}
	SetProgressFunc  func(callback func(progress interface{}))
	
	// State
	Running         bool
	CurrentTask     interface{}
	ProgressCallback func(progress interface{})
	
	// Call tracking
	ExecuteCalls      int
	StopCalls         int
	GetStatusCalls    int
	SetProgressCalls  int
}

// NewMockJobExecutor creates a new mock job executor
func NewMockJobExecutor() *MockJobExecutor {
	return &MockJobExecutor{
		Running: false,
	}
}

// Execute implements jobs.Executor
func (m *MockJobExecutor) Execute(ctx context.Context, task interface{}) error {
	m.mu.Lock()
	m.ExecuteCalls++
	m.CurrentTask = task
	m.Running = true
	m.mu.Unlock()
	
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, task)
	}
	
	// Default implementation - simulate successful execution
	defer func() {
		m.mu.Lock()
		m.Running = false
		m.CurrentTask = nil
		m.mu.Unlock()
	}()
	
	// Send some progress updates if callback is set
	if m.ProgressCallback != nil {
		// Send a generic progress update
		m.ProgressCallback(map[string]interface{}{
			"task_id":      "test-task",
			"progress":     50,
			"hash_rate":    1000000,
			"time_remaining": 3600,
			"cracked_count": 10,
		})
	}
	
	// Wait for context cancellation or return success
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Stop implements jobs.Executor
func (m *MockJobExecutor) Stop() error {
	m.mu.Lock()
	m.StopCalls++
	m.Running = false
	m.CurrentTask = nil
	m.mu.Unlock()
	
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	
	return nil
}

// GetStatus implements jobs.Executor
func (m *MockJobExecutor) GetStatus() interface{} {
	m.mu.Lock()
	m.GetStatusCalls++
	m.mu.Unlock()
	
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc()
	}
	
	// Default implementation
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"running": m.Running,
		"task_id": m.CurrentTask,
	}
}

// SetProgressCallback implements jobs.Executor
func (m *MockJobExecutor) SetProgressCallback(callback func(progress interface{})) {
	m.mu.Lock()
	m.SetProgressCalls++
	m.ProgressCallback = callback
	m.mu.Unlock()
	
	if m.SetProgressFunc != nil {
		m.SetProgressFunc(callback)
	}
}

// MockBenchmarkExecutor implements a mock benchmark executor for testing
type MockBenchmarkExecutor struct {
	mu sync.RWMutex
	
	// Control behavior
	RunBenchmarkFunc func(ctx context.Context, deviceIDs []int, hashType string) (interface{}, error)
	
	// Call tracking
	RunBenchmarkCalls int
}

// NewMockBenchmarkExecutor creates a new mock benchmark executor
func NewMockBenchmarkExecutor() *MockBenchmarkExecutor {
	return &MockBenchmarkExecutor{}
}

// RunBenchmark implements benchmark execution
func (m *MockBenchmarkExecutor) RunBenchmark(ctx context.Context, deviceIDs []int, hashType string) (interface{}, error) {
	m.mu.Lock()
	m.RunBenchmarkCalls++
	m.mu.Unlock()
	
	if m.RunBenchmarkFunc != nil {
		return m.RunBenchmarkFunc(ctx, deviceIDs, hashType)
	}
	
	// Default implementation
	return map[string]interface{}{
		"hash_type": hashType,
		"devices":   deviceIDs,
		"speed":     1000000000, // 1 GH/s
		"duration":  60,
	}, nil
}