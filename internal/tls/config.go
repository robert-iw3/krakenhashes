package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/ZerkerEOD/hashdom-backend/pkg/env"
)

// Config holds TLS configuration settings
type Config struct {
	CertFile string
	KeyFile  string
	CAFile   string
	CertsDir string
}

// NewConfig creates a new TLS configuration with default settings
// It checks environment variables first, then falls back to defaults
func NewConfig() *Config {
	// Get certificates directory from env or use default
	certsDir := env.GetOrDefault("HASHDOM_CERTS_DIR", "certs")

	// Create config with environment variables or defaults
	config := &Config{
		CertsDir: certsDir,
		CertFile: env.GetOrDefault("HASHDOM_CERT_FILE", filepath.Join(certsDir, "server.crt")),
		KeyFile:  env.GetOrDefault("HASHDOM_KEY_FILE", filepath.Join(certsDir, "server.key")),
		CAFile:   env.GetOrDefault("HASHDOM_CA_FILE", filepath.Join(certsDir, "ca.crt")),
	}

	debug.Info("TLS configuration initialized with:")
	debug.Info("  CertsDir: %s", config.CertsDir)
	debug.Info("  CertFile: %s", config.CertFile)
	debug.Info("  KeyFile: %s", config.KeyFile)
	debug.Info("  CAFile: %s", config.CAFile)

	return config
}

// LoadTLSConfig creates a TLS configuration for the server
func (c *Config) LoadTLSConfig() (*tls.Config, error) {
	debug.Info("Loading TLS configuration")

	// Load server certificate and private key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate and key: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	debug.Info("TLS configuration loaded successfully")
	return tlsConfig, nil
}

// GenerateCertificates generates server and CA certificates if they don't exist
func (c *Config) GenerateCertificates() error {
	debug.Info("Checking for existing certificates")

	// Create certs directory if it doesn't exist
	if err := os.MkdirAll(c.CertsDir, 0755); err != nil {
		return fmt.Errorf("failed to create certs directory: %v", err)
	}

	// Check if certificates already exist
	if checkFileExists(c.CertFile) && checkFileExists(c.KeyFile) && checkFileExists(c.CAFile) {
		debug.Info("Certificates already exist")
		return nil
	}

	debug.Info("Generating new certificates")
	// TODO: Implement certificate generation logic
	// This would include:
	// 1. Generating CA private key and certificate
	// 2. Generating server private key and CSR
	// 3. Signing server CSR with CA to create server certificate

	return nil
}

// checkFileExists checks if a file exists and is not a directory
func checkFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
