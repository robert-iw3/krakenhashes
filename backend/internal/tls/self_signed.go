package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// SelfSignedProvider implements the Provider interface for self-signed certificates
type SelfSignedProvider struct {
	config     *ProviderConfig
	ca         *x509.Certificate
	caKey      *rsa.PrivateKey
	cert       *x509.Certificate
	certKey    *rsa.PrivateKey
	caCertPool *x509.CertPool
	// Add client certificate fields
	clientCert *x509.Certificate
	clientKey  *rsa.PrivateKey
}

// NewSelfSignedProvider creates a new self-signed certificate provider
func NewSelfSignedProvider(config *ProviderConfig) *SelfSignedProvider {
	return &SelfSignedProvider{
		config: config,
	}
}

// Initialize sets up the self-signed certificate provider
func (p *SelfSignedProvider) Initialize() error {
	debug.Info("Initializing self-signed certificate provider")

	// Create certificates directory if it doesn't exist
	debug.Debug("Creating certificates directory: %s", p.config.CertsDir)
	if err := os.MkdirAll(p.config.CertsDir, 0755); err != nil {
		debug.Error("Failed to create certificates directory: %v", err)
		return fmt.Errorf("failed to create certificates directory: %w", err)
	}

	// Set default file paths if not provided
	if p.config.CertFile == "" {
		p.config.CertFile = filepath.Join(p.config.CertsDir, "server.crt")
		debug.Debug("Using default server certificate path: %s", p.config.CertFile)
	}
	if p.config.KeyFile == "" {
		p.config.KeyFile = filepath.Join(p.config.CertsDir, "server.key")
		debug.Debug("Using default server key path: %s", p.config.KeyFile)
	}
	if p.config.CAFile == "" {
		p.config.CAFile = filepath.Join(p.config.CertsDir, "ca.crt")
		debug.Debug("Using default CA certificate path: %s", p.config.CAFile)
	}

	// Check if certificates already exist
	if fileExists(p.config.CertFile) && fileExists(p.config.KeyFile) && fileExists(p.config.CAFile) &&
		fileExists(filepath.Join(p.config.CertsDir, "client.crt")) && fileExists(filepath.Join(p.config.CertsDir, "client.key")) {
		debug.Info("Found existing certificates, loading them")
		return p.loadExistingCertificates()
	}

	debug.Info("No existing certificates found, generating new ones")
	return p.generateNewCertificates()
}

// GetTLSConfig returns the TLS configuration for the server
func (p *SelfSignedProvider) GetTLSConfig() (*tls.Config, error) {
	debug.Debug("Getting TLS configuration")
	if p.cert == nil || p.certKey == nil {
		debug.Error("Certificates not initialized")
		return nil, fmt.Errorf("certificates not initialized")
	}

	cert := tls.Certificate{
		Certificate: [][]byte{p.cert.Raw, p.ca.Raw},
		PrivateKey:  p.certKey,
		Leaf:        p.cert,
	}

	// Create CA certificate pool if not already created
	if p.caCertPool == nil {
		debug.Debug("Creating CA certificate pool")
		p.caCertPool = x509.NewCertPool()
		p.caCertPool.AddCert(p.ca)
	}

	debug.Debug("Creating TLS configuration with secure defaults")
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    p.caCertPool,
		// Make client certificates optional by default
		ClientAuth: tls.VerifyClientCertIfGiven,
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		// Add verification callback for client certs
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				debug.Debug("No client certificate provided")
				return nil
			}

			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				debug.Error("Failed to parse client certificate: %v", err)
				return fmt.Errorf("failed to parse client certificate: %w", err)
			}

			// Log certificate details
			debug.Debug("Verifying client certificate:")
			debug.Debug("- Subject: %s", cert.Subject)
			debug.Debug("- Issuer: %s", cert.Issuer)
			debug.Debug("- Serial: %s", cert.SerialNumber)
			debug.Debug("- Not Before: %s", cert.NotBefore)
			debug.Debug("- Not After: %s", cert.NotAfter)

			// Verify certificate is not expired
			if time.Now().After(cert.NotAfter) || time.Now().Before(cert.NotBefore) {
				debug.Error("Client certificate is not valid at this time")
				return fmt.Errorf("client certificate expired or not yet valid")
			}

			// Verify certificate was issued by our CA
			opts := x509.VerifyOptions{
				Roots:     p.caCertPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}

			if _, err := cert.Verify(opts); err != nil {
				debug.Error("Client certificate verification failed: %v", err)
				return fmt.Errorf("client certificate verification failed: %w", err)
			}

			debug.Info("Client certificate successfully verified")
			return nil
		},
	}, nil
}

// GetCACertPool returns the CA certificate pool
func (p *SelfSignedProvider) GetCACertPool() (*x509.CertPool, error) {
	debug.Debug("Getting CA certificate pool")
	if p.caCertPool == nil {
		debug.Error("CA certificate pool not initialized")
		return nil, fmt.Errorf("CA certificate pool not initialized")
	}
	return p.caCertPool, nil
}

// Cleanup performs any necessary cleanup
func (p *SelfSignedProvider) Cleanup() error {
	// No cleanup needed for self-signed certificates
	return nil
}

// loadExistingCertificates loads existing certificates from disk
func (p *SelfSignedProvider) loadExistingCertificates() error {
	debug.Info("Loading existing certificates")

	// Load CA certificate
	debug.Debug("Loading CA certificate from: %s", p.config.CAFile)
	caCertPEM, err := os.ReadFile(p.config.CAFile)
	if err != nil {
		debug.Error("Failed to read CA certificate: %v", err)
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		debug.Error("Failed to decode CA certificate PEM")
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	p.ca, err = x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse CA certificate: %v", err)
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Load CA private key
	debug.Debug("Loading CA private key from: %s", filepath.Join(p.config.CertsDir, "ca.key"))
	caKeyPEM, err := os.ReadFile(filepath.Join(p.config.CertsDir, "ca.key"))
	if err != nil {
		debug.Error("Failed to read CA private key: %v", err)
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		debug.Error("Failed to decode CA private key PEM")
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	p.caKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse CA private key: %v", err)
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	// Verify CA key matches certificate
	if !p.ca.PublicKey.(*rsa.PublicKey).Equal(&p.caKey.PublicKey) {
		debug.Error("CA private key does not match certificate")
		return fmt.Errorf("CA private key does not match certificate")
	}

	debug.Info("Successfully loaded CA certificate and private key")

	// Load server certificate
	debug.Debug("Loading server certificate from: %s", p.config.CertFile)
	certPEM, err := os.ReadFile(p.config.CertFile)
	if err != nil {
		debug.Error("Failed to read server certificate: %v", err)
		return fmt.Errorf("failed to read server certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		debug.Error("Failed to decode server certificate PEM")
		return fmt.Errorf("failed to decode server certificate PEM")
	}

	p.cert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse server certificate: %v", err)
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Load server private key
	debug.Debug("Loading server private key from: %s", p.config.KeyFile)
	keyPEM, err := os.ReadFile(p.config.KeyFile)
	if err != nil {
		debug.Error("Failed to read server private key: %v", err)
		return fmt.Errorf("failed to read server private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		debug.Error("Failed to decode server private key PEM")
		return fmt.Errorf("failed to decode server private key PEM")
	}

	p.certKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse server private key: %v", err)
		return fmt.Errorf("failed to parse server private key: %w", err)
	}

	// Create CA certificate pool
	debug.Debug("Creating CA certificate pool")
	p.caCertPool = x509.NewCertPool()
	p.caCertPool.AddCert(p.ca)

	// Load client certificate
	debug.Debug("Loading client certificate from: %s", filepath.Join(p.config.CertsDir, "client.crt"))
	clientCertPEM, err := os.ReadFile(filepath.Join(p.config.CertsDir, "client.crt"))
	if err != nil {
		debug.Error("Failed to read client certificate: %v", err)
		return fmt.Errorf("failed to read client certificate: %w", err)
	}

	clientCertBlock, _ := pem.Decode(clientCertPEM)
	if clientCertBlock == nil {
		debug.Error("Failed to decode client certificate PEM")
		return fmt.Errorf("failed to decode client certificate PEM")
	}

	p.clientCert, err = x509.ParseCertificate(clientCertBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse client certificate: %v", err)
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Load client private key
	debug.Debug("Loading client private key from: %s", filepath.Join(p.config.CertsDir, "client.key"))
	clientKeyPEM, err := os.ReadFile(filepath.Join(p.config.CertsDir, "client.key"))
	if err != nil {
		debug.Error("Failed to read client private key: %v", err)
		return fmt.Errorf("failed to read client private key: %w", err)
	}

	clientKeyBlock, _ := pem.Decode(clientKeyPEM)
	if clientKeyBlock == nil {
		debug.Error("Failed to decode client private key PEM")
		return fmt.Errorf("failed to decode client private key PEM")
	}

	p.clientKey, err = x509.ParsePKCS1PrivateKey(clientKeyBlock.Bytes)
	if err != nil {
		debug.Error("Failed to parse client private key: %v", err)
		return fmt.Errorf("failed to parse client private key: %w", err)
	}

	debug.Info("Successfully loaded existing certificates")
	return nil
}

// generateNewCertificates generates new CA and server certificates
func (p *SelfSignedProvider) generateNewCertificates() error {
	debug.Info("Generating new certificates")

	// Generate CA key pair
	debug.Debug("Generating CA key pair with key size: %d", p.config.KeySize)
	caKey, err := rsa.GenerateKey(rand.Reader, p.config.KeySize)
	if err != nil {
		debug.Error("Failed to generate CA private key: %v", err)
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}
	p.caKey = caKey

	// Create CA certificate
	debug.Debug("Creating CA certificate with validity: %d days", p.config.Validity.CA)
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{p.config.CADetails.Country},
			Organization:       []string{p.config.CADetails.Organization},
			OrganizationalUnit: []string{p.config.CADetails.OrganizationalUnit},
			CommonName:         p.config.CADetails.CommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, p.config.Validity.CA),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign the CA certificate
	debug.Debug("Self-signing CA certificate")
	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		debug.Error("Failed to create CA certificate: %v", err)
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	p.ca, err = x509.ParseCertificate(caBytes)
	if err != nil {
		debug.Error("Failed to parse CA certificate: %v", err)
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Generate server certificate
	debug.Debug("Generating server key pair with key size: %d", p.config.KeySize)
	serverKey, err := rsa.GenerateKey(rand.Reader, p.config.KeySize)
	if err != nil {
		debug.Error("Failed to generate server private key: %v", err)
		return fmt.Errorf("failed to generate server private key: %w", err)
	}
	p.certKey = serverKey

	// Create server certificate
	debug.Debug("Creating server certificate with validity: %d days", p.config.Validity.Server)

	// Start with default DNS names
	dnsNames := []string{
		"localhost",
		"127.0.0.1",
		p.config.CADetails.CommonName,
	}
	// Add additional DNS names from config
	dnsNames = append(dnsNames, p.config.AdditionalDNSNames...)

	// Start with default IP addresses
	ipAddresses := []net.IP{
		net.ParseIP("127.0.0.1"),
	}
	// Add additional IP addresses from config
	for _, ipStr := range p.config.AdditionalIPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			debug.Warning("Invalid IP address in config: %s", ipStr)
		}
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Country:            []string{p.config.CADetails.Country},
			Organization:       []string{p.config.CADetails.Organization},
			OrganizationalUnit: []string{"HashDom Server"},
			CommonName:         "HashDom Server",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, p.config.Validity.Server),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	// Sign the server certificate with CA
	debug.Debug("Signing server certificate with CA")
	serverBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, p.ca, &serverKey.PublicKey, caKey)
	if err != nil {
		debug.Error("Failed to create server certificate: %v", err)
		return fmt.Errorf("failed to create server certificate: %w", err)
	}

	p.cert, err = x509.ParseCertificate(serverBytes)
	if err != nil {
		debug.Error("Failed to parse server certificate: %v", err)
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Generate client certificate
	debug.Debug("Generating shared client key pair with key size: %d", p.config.KeySize)
	clientKey, err := rsa.GenerateKey(rand.Reader, p.config.KeySize)
	if err != nil {
		debug.Error("Failed to generate client private key: %v", err)
		return fmt.Errorf("failed to generate client private key: %w", err)
	}
	p.clientKey = clientKey

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Country:            []string{p.config.CADetails.Country},
			Organization:       []string{p.config.CADetails.Organization},
			OrganizationalUnit: []string{"HashDom Client"},
			CommonName:         "HashDom Client",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, p.config.Validity.Server),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the client certificate with CA
	debug.Debug("Signing client certificate with CA")
	clientBytes, err := x509.CreateCertificate(rand.Reader, clientTemplate, p.ca, &clientKey.PublicKey, caKey)
	if err != nil {
		debug.Error("Failed to create client certificate: %v", err)
		return fmt.Errorf("failed to create client certificate: %w", err)
	}

	p.clientCert, err = x509.ParseCertificate(clientBytes)
	if err != nil {
		debug.Error("Failed to parse client certificate: %v", err)
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Save all certificates
	debug.Info("Saving certificates to disk")
	if err := p.saveCertificates(); err != nil {
		debug.Error("Failed to save certificates: %v", err)
		return fmt.Errorf("failed to save certificates: %w", err)
	}

	debug.Info("Successfully generated and saved all certificates")
	return nil
}

// saveCertificates saves the certificates to disk
func (p *SelfSignedProvider) saveCertificates() error {
	// Save CA certificate
	debug.Debug("Saving CA certificate to: %s", p.config.CAFile)
	caCertFile, err := os.Create(p.config.CAFile)
	if err != nil {
		debug.Error("Failed to create CA certificate file: %v", err)
		return fmt.Errorf("failed to create CA certificate file: %w", err)
	}
	defer caCertFile.Close()

	if err := pem.Encode(caCertFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.ca.Raw,
	}); err != nil {
		debug.Error("Failed to write CA certificate: %v", err)
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Save CA private key
	debug.Debug("Saving CA private key to: %s", filepath.Join(p.config.CertsDir, "ca.key"))
	caKeyFile, err := os.OpenFile(filepath.Join(p.config.CertsDir, "ca.key"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		debug.Error("Failed to create CA private key file: %v", err)
		return fmt.Errorf("failed to create CA private key file: %w", err)
	}
	defer caKeyFile.Close()

	if err := pem.Encode(caKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(p.caKey),
	}); err != nil {
		debug.Error("Failed to write CA private key: %v", err)
		return fmt.Errorf("failed to write CA private key: %w", err)
	}

	// Save server certificate
	debug.Debug("Saving server certificate to: %s", p.config.CertFile)
	certFile, err := os.Create(p.config.CertFile)
	if err != nil {
		debug.Error("Failed to create server certificate file: %v", err)
		return fmt.Errorf("failed to create server certificate file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.cert.Raw,
	}); err != nil {
		debug.Error("Failed to write server certificate: %v", err)
		return fmt.Errorf("failed to write server certificate: %w", err)
	}

	// Save server private key
	debug.Debug("Saving server private key to: %s", p.config.KeyFile)
	keyFile, err := os.OpenFile(p.config.KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		debug.Error("Failed to create server private key file: %v", err)
		return fmt.Errorf("failed to create server private key file: %w", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(p.certKey),
	}); err != nil {
		debug.Error("Failed to write server private key: %v", err)
		return fmt.Errorf("failed to write server private key: %w", err)
	}

	// Save client certificate
	debug.Debug("Saving client certificate to: %s", filepath.Join(p.config.CertsDir, "client.crt"))
	clientCertFile, err := os.Create(filepath.Join(p.config.CertsDir, "client.crt"))
	if err != nil {
		debug.Error("Failed to create client certificate file: %v", err)
		return fmt.Errorf("failed to create client certificate file: %w", err)
	}
	defer clientCertFile.Close()

	if err := pem.Encode(clientCertFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.clientCert.Raw,
	}); err != nil {
		debug.Error("Failed to write client certificate: %v", err)
		return fmt.Errorf("failed to write client certificate: %w", err)
	}

	// Save client private key
	debug.Debug("Saving client private key to: %s", filepath.Join(p.config.CertsDir, "client.key"))
	clientKeyFile, err := os.OpenFile(filepath.Join(p.config.CertsDir, "client.key"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		debug.Error("Failed to create client private key file: %v", err)
		return fmt.Errorf("failed to create client private key file: %w", err)
	}
	defer clientKeyFile.Close()

	if err := pem.Encode(clientKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(p.clientKey),
	}); err != nil {
		debug.Error("Failed to write client private key: %v", err)
		return fmt.Errorf("failed to write client private key: %w", err)
	}

	debug.Info("Successfully saved all certificates")
	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ExportCACertificate exports the CA certificate in PEM format for browsers
func (p *SelfSignedProvider) ExportCACertificate() ([]byte, error) {
	debug.Info("Exporting CA certificate for browsers")
	if p.ca == nil {
		debug.Error("CA certificate not initialized")
		return nil, fmt.Errorf("CA certificate not initialized")
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.ca.Raw,
	}

	// Encode to PEM format
	pemData := pem.EncodeToMemory(pemBlock)
	if pemData == nil {
		debug.Error("Failed to encode CA certificate to PEM")
		return nil, fmt.Errorf("failed to encode CA certificate")
	}

	debug.Info("Successfully exported CA certificate")
	return pemData, nil
}

// ExportCACertificateToFile exports the CA certificate to a file for distribution
func (p *SelfSignedProvider) ExportCACertificateToFile(path string) error {
	debug.Info("Exporting CA certificate to file: %s", path)

	// Get PEM data
	pemData, err := p.ExportCACertificate()
	if err != nil {
		return fmt.Errorf("failed to export CA certificate: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		debug.Error("Failed to create directory for CA certificate: %v", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, pemData, 0644); err != nil {
		debug.Error("Failed to write CA certificate to file: %v", err)
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	debug.Info("Successfully exported CA certificate to file")
	return nil
}

// GenerateClientCertificate generates a client certificate signed by the CA
func (p *SelfSignedProvider) GenerateClientCertificate(commonName string) (*tls.Certificate, error) {
	debug.Info("Generating client certificate for: %s", commonName)

	// Ensure CA key is loaded
	if p.caKey == nil {
		debug.Error("CA private key not initialized")
		return nil, fmt.Errorf("CA private key not initialized")
	}

	// Generate client key pair
	debug.Debug("Generating client key pair with key size: %d", p.config.KeySize)
	clientKey, err := rsa.GenerateKey(rand.Reader, p.config.KeySize)
	if err != nil {
		debug.Error("Failed to generate client key pair: %v", err)
		return nil, fmt.Errorf("failed to generate client key pair: %w", err)
	}

	// Create client certificate template
	debug.Debug("Creating client certificate template")
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Country:            []string{p.config.CADetails.Country},
			Organization:       []string{p.config.CADetails.Organization},
			OrganizationalUnit: []string{"HashDom Agents"},
			CommonName:         commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, p.config.Validity.Server),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the client certificate with CA
	debug.Debug("Signing client certificate with CA")
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientTemplate, p.ca, &clientKey.PublicKey, p.caKey)
	if err != nil {
		debug.Error("Failed to create client certificate: %v", err)
		return nil, fmt.Errorf("failed to create client certificate: %w", err)
	}

	// Encode certificate to PEM
	debug.Debug("Encoding client certificate to PEM")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCertBytes,
	})
	if certPEM == nil {
		debug.Error("Failed to encode client certificate to PEM")
		return nil, fmt.Errorf("failed to encode client certificate")
	}

	// Encode private key to PEM
	debug.Debug("Encoding client private key to PEM")
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
	})
	if keyPEM == nil {
		debug.Error("Failed to encode client private key to PEM")
		return nil, fmt.Errorf("failed to encode client private key")
	}

	debug.Info("Successfully generated client certificate")
	return &tls.Certificate{
		Certificate: [][]byte{certPEM},
		PrivateKey:  keyPEM,
	}, nil
}

// GetClientCertificate returns the client certificate and private key in PEM format
func (p *SelfSignedProvider) GetClientCertificate() ([]byte, []byte, error) {
	debug.Info("Exporting client certificate and key")
	if p.clientCert == nil || p.clientKey == nil {
		debug.Error("Client certificate not initialized")
		return nil, nil, fmt.Errorf("client certificate not initialized")
	}

	// Create certificate PEM block
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.clientCert.Raw,
	})
	if certPEM == nil {
		debug.Error("Failed to encode client certificate to PEM")
		return nil, nil, fmt.Errorf("failed to encode client certificate")
	}

	// Create private key PEM block
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(p.clientKey),
	})
	if keyPEM == nil {
		debug.Error("Failed to encode client private key to PEM")
		return nil, nil, fmt.Errorf("failed to encode client private key")
	}

	debug.Info("Successfully exported client certificate and key")
	return certPEM, keyPEM, nil
}
