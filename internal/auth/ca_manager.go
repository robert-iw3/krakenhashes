package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// CAManager handles certificate authority operations
type CAManager struct {
	caCertPath string
	caKeyPath  string
	caCert     *x509.Certificate
	caKey      *rsa.PrivateKey
}

// NewCAManager creates a new instance of CAManager
func NewCAManager(caCertPath, caKeyPath string) (*CAManager, error) {
	manager := &CAManager{
		caCertPath: caCertPath,
		caKeyPath:  caKeyPath,
	}

	// Load CA certificate and key
	if err := manager.loadCA(); err != nil {
		return nil, fmt.Errorf("failed to load CA: %w", err)
	}

	return manager, nil
}

// loadCA loads the CA certificate and private key from disk
func (m *CAManager) loadCA() error {
	// Read CA certificate
	certPEM, err := os.ReadFile(m.caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Read CA private key
	keyPEM, err := os.ReadFile(m.caKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	m.caCert = cert
	m.caKey = key

	return nil
}

// GenerateAgentCertificate generates a new certificate for an agent
func (m *CAManager) GenerateAgentCertificate(ctx context.Context, agentID string) ([]byte, []byte, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: generateSerialNumber(),
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("agent-%s", agentID),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &privateKey.PublicKey, m.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// GetCACertificate returns the CA certificate
func (m *CAManager) GetCACertificate() ([]byte, error) {
	return os.ReadFile(m.caCertPath)
}

// generateSerialNumber generates a random serial number for certificates
func generateSerialNumber() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(fmt.Sprintf("failed to generate serial number: %v", err))
	}
	return serialNumber
}
