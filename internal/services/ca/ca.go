package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// CA represents the Certificate Authority
type CA struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

// Config holds CA configuration
type Config struct {
	Country            string
	Organization       string
	OrganizationalUnit string
	CommonName         string
}

// LoadConfigFromEnv loads CA configuration from environment variables
func LoadConfigFromEnv() Config {
	return Config{
		Country:            getEnvOrDefault("CA_COUNTRY", "US"),
		Organization:       getEnvOrDefault("CA_ORGANIZATION", "HashDom"),
		OrganizationalUnit: getEnvOrDefault("CA_ORGANIZATIONAL_UNIT", "HashDom CA"),
		CommonName:         getEnvOrDefault("CA_COMMON_NAME", "HashDom Root CA"),
	}
}

// New creates a new Certificate Authority
func New(config Config) (*CA, error) {
	// Generate CA private key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		debug.Error("failed to generate CA private key: %v", err)
		return nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		debug.Error("failed to generate serial number: %v", err)
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{config.Country},
			Organization:       []string{config.Organization},
			OrganizationalUnit: []string{config.OrganizationalUnit},
			CommonName:         config.CommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0), // 100 years - effectively no expiration
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0, // No intermediate CAs allowed
	}

	// Self-sign the CA certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		debug.Error("failed to create CA certificate: %v", err)
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		debug.Error("failed to parse CA certificate: %v", err)
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &CA{
		cert: cert,
		key:  key,
	}, nil
}

// IssueCertificate issues a new certificate for an agent
func (ca *CA) IssueCertificate(commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate key pair for the agent
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		debug.Error("failed to generate agent key: %v", err)
		return nil, nil, fmt.Errorf("failed to generate agent key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		debug.Error("failed to generate serial number: %v", err)
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0), // 100 years - effectively no expiration
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		debug.Error("failed to create agent certificate: %v", err)
		return nil, nil, fmt.Errorf("failed to create agent certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		debug.Error("failed to parse agent certificate: %v", err)
		return nil, nil, fmt.Errorf("failed to parse agent certificate: %w", err)
	}

	return cert, key, nil
}

// EncodeCertificate encodes a certificate to PEM format
func EncodeCertificate(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// EncodePrivateKey encodes a private key to PEM format
func EncodePrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// GetCACertificate returns the CA certificate in PEM format
func (ca *CA) GetCACertificate() []byte {
	return EncodeCertificate(ca.cert)
}

// VerifyCertificate verifies if a certificate was issued by this CA
func (ca *CA) VerifyCertificate(cert *x509.Certificate) error {
	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		debug.Error("certificate verification failed: %v", err)
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
