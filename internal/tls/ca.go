package tls

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
	"github.com/ZerkerEOD/hashdom-backend/pkg/env"
)

// CADetails holds the configuration for CA certificate generation
type CADetails struct {
	Country            string
	Organization       string
	OrganizationalUnit string
	CommonName         string
}

// CA represents a Certificate Authority
type CA struct {
	cert    *x509.Certificate
	key     *rsa.PrivateKey
	details CADetails
}

// NewCA creates a new Certificate Authority instance
func NewCA() (*CA, error) {
	debug.Info("Creating new CA instance")

	// Load CA details from environment
	details := CADetails{
		Country:            env.GetOrDefault("CA_COUNTRY", "US"),
		Organization:       env.GetOrDefault("CA_ORGANIZATION", "HashDom"),
		OrganizationalUnit: env.GetOrDefault("CA_ORGANIZATIONAL_UNIT", "HashDom CA"),
		CommonName:         env.GetOrDefault("CA_COMMON_NAME", "HashDom Root CA"),
	}

	debug.Info("CA details loaded from environment:")
	debug.Info("  Country: %s", details.Country)
	debug.Info("  Organization: %s", details.Organization)
	debug.Info("  OrganizationalUnit: %s", details.OrganizationalUnit)
	debug.Info("  CommonName: %s", details.CommonName)

	return &CA{details: details}, nil
}

// LoadOrCreate loads an existing CA or creates a new one
func (ca *CA) LoadOrCreate(certFile, keyFile string) error {
	debug.Info("Loading or creating CA")

	// Check if CA files exist
	if exists(certFile) && exists(keyFile) {
		return ca.loadExisting(certFile, keyFile)
	}

	return ca.createNew(certFile, keyFile)
}

// loadExisting loads an existing CA from files
func (ca *CA) loadExisting(certFile, keyFile string) error {
	debug.Info("Loading existing CA from files")

	// Read certificate
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %v", err)
	}

	ca.cert = cert
	ca.key = key
	debug.Info("Successfully loaded existing CA")
	return nil
}

// createNew creates a new CA and saves it to files
func (ca *CA) createNew(certFile, keyFile string) error {
	debug.Info("Creating new CA")

	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %v", err)
	}

	// Prepare certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{ca.details.Country},
			Organization:       []string{ca.details.Organization},
			OrganizationalUnit: []string{ca.details.OrganizationalUnit},
			CommonName:         ca.details.CommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years validity
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Save certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return fmt.Errorf("failed to write CA certificate: %v", err)
	}

	// Save private key
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create CA private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return fmt.Errorf("failed to write CA private key: %v", err)
	}

	// Parse the created certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("failed to parse created CA certificate: %v", err)
	}

	ca.cert = cert
	ca.key = key
	debug.Info("Successfully created new CA")
	return nil
}

// SignCertificate signs a CSR and returns a signed certificate
func (ca *CA) SignCertificate(csr *x509.CertificateRequest) (*x509.Certificate, error) {
	debug.Info("Signing certificate request")

	if ca.cert == nil || ca.key == nil {
		return nil, fmt.Errorf("CA not initialized")
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0), // 1 year validity
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca.cert, csr.PublicKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created certificate: %v", err)
	}

	debug.Info("Successfully signed certificate")
	return cert, nil
}

// exists checks if a file exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
