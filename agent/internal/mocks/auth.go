package mocks

import (
	"sync"
)

// MockAPIKeyManager implements a mock API key manager for testing
type MockAPIKeyManager struct {
	mu sync.RWMutex
	
	// Control behavior
	GetAPIKeyFunc      func() string
	SetAPIKeyFunc      func(apiKey string) error
	HasAPIKeyFunc      func() bool
	ClearAPIKeyFunc    func() error
	GetAgentIDFunc     func() string
	SetAgentIDFunc     func(agentID string) error
	
	// State
	APIKey  string
	AgentID string
	
	// Call tracking
	GetAPIKeyCalls   int
	SetAPIKeyCalls   int
	HasAPIKeyCalls   int
	ClearAPIKeyCalls int
	GetAgentIDCalls  int
	SetAgentIDCalls  int
}

// NewMockAPIKeyManager creates a new mock API key manager
func NewMockAPIKeyManager() *MockAPIKeyManager {
	return &MockAPIKeyManager{}
}

// GetAPIKey implements auth.APIKeyManager
func (m *MockAPIKeyManager) GetAPIKey() string {
	m.mu.Lock()
	m.GetAPIKeyCalls++
	m.mu.Unlock()
	
	if m.GetAPIKeyFunc != nil {
		return m.GetAPIKeyFunc()
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.APIKey
}

// SetAPIKey implements auth.APIKeyManager
func (m *MockAPIKeyManager) SetAPIKey(apiKey string) error {
	m.mu.Lock()
	m.SetAPIKeyCalls++
	m.mu.Unlock()
	
	if m.SetAPIKeyFunc != nil {
		return m.SetAPIKeyFunc(apiKey)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.APIKey = apiKey
	return nil
}

// HasAPIKey implements auth.APIKeyManager
func (m *MockAPIKeyManager) HasAPIKey() bool {
	m.mu.Lock()
	m.HasAPIKeyCalls++
	m.mu.Unlock()
	
	if m.HasAPIKeyFunc != nil {
		return m.HasAPIKeyFunc()
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.APIKey != ""
}

// ClearAPIKey implements auth.APIKeyManager
func (m *MockAPIKeyManager) ClearAPIKey() error {
	m.mu.Lock()
	m.ClearAPIKeyCalls++
	m.mu.Unlock()
	
	if m.ClearAPIKeyFunc != nil {
		return m.ClearAPIKeyFunc()
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.APIKey = ""
	return nil
}

// GetAgentID implements auth.APIKeyManager
func (m *MockAPIKeyManager) GetAgentID() string {
	m.mu.Lock()
	m.GetAgentIDCalls++
	m.mu.Unlock()
	
	if m.GetAgentIDFunc != nil {
		return m.GetAgentIDFunc()
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.AgentID
}

// SetAgentID implements auth.APIKeyManager
func (m *MockAPIKeyManager) SetAgentID(agentID string) error {
	m.mu.Lock()
	m.SetAgentIDCalls++
	m.mu.Unlock()
	
	if m.SetAgentIDFunc != nil {
		return m.SetAgentIDFunc(agentID)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AgentID = agentID
	return nil
}

// MockRegistrationClient implements a mock registration client for testing
type MockRegistrationClient struct {
	mu sync.RWMutex
	
	// Control behavior
	RegisterFunc func(claimCode string, hostname string) (interface{}, error)
	
	// Call tracking
	RegisterCalls int
	LastClaimCode string
	LastHostname  string
}

// NewMockRegistrationClient creates a new mock registration client
func NewMockRegistrationClient() *MockRegistrationClient {
	return &MockRegistrationClient{}
}

// Register implements registration functionality
func (m *MockRegistrationClient) Register(claimCode string, hostname string) (interface{}, error) {
	m.mu.Lock()
	m.RegisterCalls++
	m.LastClaimCode = claimCode
	m.LastHostname = hostname
	m.mu.Unlock()
	
	if m.RegisterFunc != nil {
		return m.RegisterFunc(claimCode, hostname)
	}
	
	// Default implementation - return a map like the real response
	return map[string]interface{}{
		"agent_id": 123,
		"api_key":  "test-api-key",
		"endpoints": map[string]string{
			"websocket": "wss://test.example.com/ws",
		},
	}, nil
}