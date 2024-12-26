package ca

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// Manager handles CA lifecycle and persistence
type Manager struct {
	ca        *CA
	certFile  string
	keyFile   string
	initMutex sync.Mutex
}

// NewManager creates a new CA manager
func NewManager() *Manager {
	// Get certificate paths from environment
	certFile := getEnvOrDefault("CA_CERT_PATH", "/etc/hashdom/ca/ca.crt")
	keyFile := getEnvOrDefault("CA_KEY_PATH", "/etc/hashdom/ca/ca.key")

	// If paths are relative, make them relative to current working directory
	if !filepath.IsAbs(certFile) || !filepath.IsAbs(keyFile) {
		cwd, err := os.Getwd()
		if err == nil {
			if !filepath.IsAbs(certFile) {
				certFile = filepath.Join(cwd, certFile)
			}
			if !filepath.IsAbs(keyFile) {
				keyFile = filepath.Join(cwd, keyFile)
			}
		}
	}

	// Ensure the CA directory exists with restricted permissions
	caDir := filepath.Dir(certFile)
	if err := os.MkdirAll(caDir, 0700); err != nil {
		debug.Error("failed to create CA directory: %v", err)
	}

	return &Manager{
		certFile: certFile,
		keyFile:  keyFile,
	}
}

// GetCA returns the CA instance, initializing it if necessary
func (m *Manager) GetCA() (*CA, error) {
	m.initMutex.Lock()
	defer m.initMutex.Unlock()

	if m.ca != nil {
		return m.ca, nil
	}

	// Try to load existing CA
	if ca, err := m.loadCA(); err == nil {
		m.ca = ca
		return m.ca, nil
	}

	// Create new CA if none exists
	config := LoadConfigFromEnv()
	ca, err := New(config)
	if err != nil {
		debug.Error("failed to create new CA: %v", err)
		return nil, fmt.Errorf("failed to create new CA: %w", err)
	}

	// Save the new CA
	if err := m.saveCA(ca); err != nil {
		debug.Error("failed to save new CA: %v", err)
		return nil, fmt.Errorf("failed to save new CA: %w", err)
	}

	m.ca = ca
	return m.ca, nil
}

// loadCA loads the CA from disk
func (m *Manager) loadCA() (*CA, error) {
	// Read certificate
	certPEM, err := os.ReadFile(m.certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(m.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA private key: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse private key
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode CA private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	return &CA{
		cert: cert,
		key:  key,
	}, nil
}

// saveCA saves the CA to disk with restricted permissions
func (m *Manager) saveCA(ca *CA) error {
	// Create directory if it doesn't exist
	caDir := filepath.Dir(m.certFile)
	if err := os.MkdirAll(caDir, 0700); err != nil {
		return fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Save certificate (readable by others since it's public)
	certPEM := EncodeCertificate(ca.cert)
	if err := os.WriteFile(m.certFile, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Save private key (restricted access)
	keyPEM := EncodePrivateKey(ca.key)
	if err := os.WriteFile(m.keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write CA private key: %w", err)
	}

	return nil
}

// GetCACertificate returns the CA certificate in PEM format
func (m *Manager) GetCACertificate() ([]byte, error) {
	ca, err := m.GetCA()
	if err != nil {
		return nil, err
	}
	return ca.GetCACertificate(), nil
}

// IssueCertificate issues a new certificate using the CA
func (m *Manager) IssueCertificate(commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	ca, err := m.GetCA()
	if err != nil {
		return nil, nil, err
	}
	return ca.IssueCertificate(commonName)
}
