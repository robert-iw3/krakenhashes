package testutil

import (
	"context"
	"fmt"
	"sync"
)

// MockEmailService is a mock implementation of the email service
type MockEmailService struct {
	mu            sync.Mutex
	SentCodes     map[string]string // email -> code mapping
	SendError     error             // Error to return from SendMFACode
	CallCount     int
	LastRecipient string
	LastCode      string
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		SentCodes: make(map[string]string),
	}
}

// SendMFACode implements the EmailService interface
func (m *MockEmailService) SendMFACode(ctx context.Context, to string, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastRecipient = to
	m.LastCode = code

	if m.SendError != nil {
		return m.SendError
	}

	m.SentCodes[to] = code
	return nil
}

// GetSentCode returns the code sent to a specific email
func (m *MockEmailService) GetSentCode(email string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	code, ok := m.SentCodes[email]
	return code, ok
}

// Reset clears all sent codes and counters
func (m *MockEmailService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SentCodes = make(map[string]string)
	m.SendError = nil
	m.CallCount = 0
	m.LastRecipient = ""
	m.LastCode = ""
}

// SetSendError sets an error to be returned on the next SendMFACode call
func (m *MockEmailService) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendError = err
}

// MockTLSProvider is a mock implementation of the TLS provider
type MockTLSProvider struct {
	ClientCertPEM string
	ClientKeyPEM  string
	CACertPEM     string
	GetClientErr  error
	GetCAErr      error
}

// GetClientCertificate returns mock client certificate and key
func (m *MockTLSProvider) GetClientCertificate() ([]byte, []byte, error) {
	if m.GetClientErr != nil {
		return nil, nil, m.GetClientErr
	}
	return []byte(m.ClientCertPEM), []byte(m.ClientKeyPEM), nil
}

// ExportCACertificate returns mock CA certificate
func (m *MockTLSProvider) ExportCACertificate() ([]byte, error) {
	if m.GetCAErr != nil {
		return nil, m.GetCAErr
	}
	return []byte(m.CACertPEM), nil
}

// GetCertificate is not used in tests
func (m *MockTLSProvider) GetCertificate() (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetKey is not used in tests
func (m *MockTLSProvider) GetKey() (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}
