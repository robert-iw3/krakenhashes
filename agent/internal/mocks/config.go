package mocks

import (
	"sync"
	
	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
)

// MockConfigManager implements a mock configuration manager for testing
type MockConfigManager struct {
	mu sync.RWMutex
	
	// Control behavior
	GetConfigFunc    func() *config.Config
	SaveConfigFunc   func(cfg *config.Config) error
	LoadConfigFunc   func() error
	
	// State
	Config *config.Config
	
	// Call tracking
	GetConfigCalls  int
	SaveConfigCalls int
	LoadConfigCalls int
}

// NewMockConfigManager creates a new mock config manager
func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		Config: &config.Config{
			DataDirectory:      "/tmp/test-data",
			HashcatExtraParams: "",
		},
	}
}

// GetConfig implements config management
func (m *MockConfigManager) GetConfig() *config.Config {
	m.mu.Lock()
	m.GetConfigCalls++
	m.mu.Unlock()
	
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc()
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Config
}

// SaveConfig implements config management
func (m *MockConfigManager) SaveConfig(cfg *config.Config) error {
	m.mu.Lock()
	m.SaveConfigCalls++
	m.mu.Unlock()
	
	if m.SaveConfigFunc != nil {
		return m.SaveConfigFunc(cfg)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Config = cfg
	return nil
}

// LoadConfig implements config management
func (m *MockConfigManager) LoadConfig() error {
	m.mu.Lock()
	m.LoadConfigCalls++
	m.mu.Unlock()
	
	if m.LoadConfigFunc != nil {
		return m.LoadConfigFunc()
	}
	
	// Default implementation - no-op
	return nil
}

// SetConfig is a helper method to set the config for testing
func (m *MockConfigManager) SetConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Config = cfg
}