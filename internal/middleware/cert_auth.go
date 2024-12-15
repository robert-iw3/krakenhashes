package middleware

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/yourusername/hashdom/internal/repository"
	"github.com/yourusername/hashdom/internal/services/ca"
	"github.com/yourusername/hashdom/pkg/debug"
)

// CertAuth handles certificate-based authentication for agents
type CertAuth struct {
	agentRepo *repository.AgentRepository
	ca        *ca.CA
}

// NewCertAuth creates a new certificate authentication middleware
func NewCertAuth(agentRepo *repository.AgentRepository, ca *ca.CA) *CertAuth {
	return &CertAuth{
		agentRepo: agentRepo,
		ca:        ca,
	}
}

// Middleware returns the certificate authentication middleware handler
func (ca *CertAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client certificate
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			debug.Error("no client certificate provided")
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		cert := r.TLS.PeerCertificates[0]

		// Verify certificate was issued by our CA
		if err := ca.ca.VerifyCertificate(cert); err != nil {
			debug.Error("certificate verification failed: %v", err)
			http.Error(w, "Invalid certificate", http.StatusUnauthorized)
			return
		}

		// Get agent by certificate
		certPEM := ca.EncodeCertificate(cert)
		agent, err := ca.agentRepo.GetByCertificate(r.Context(), string(certPEM))
		if err != nil {
			debug.Error("failed to get agent by certificate: %v", err)
			http.Error(w, "Invalid certificate", http.StatusUnauthorized)
			return
		}

		// Store agent in context
		ctx := context.WithValue(r.Context(), "agent", agent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateAgentCertificate generates a new certificate for an agent
func (ca *CertAuth) GenerateAgentCertificate(agentID uint, commonName string) error {
	// Generate new certificate
	cert, key, err := ca.ca.IssueCertificate(commonName)
	if err != nil {
		debug.Error("failed to issue certificate: %v", err)
		return fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Encode certificate and key
	certPEM := ca.EncodeCertificate(cert)
	keyPEM := ca.EncodePrivateKey(key)

	// Update agent with new certificate
	if err := ca.agentRepo.UpdateCertificate(context.Background(), agentID, string(certPEM)); err != nil {
		debug.Error("failed to update agent certificate: %v", err)
		return fmt.Errorf("failed to update agent certificate: %w", err)
	}

	return nil
}

// EncodeCertificate encodes a certificate to PEM format
func (ca *CertAuth) EncodeCertificate(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// GetCACertificate returns the CA certificate in PEM format
func (ca *CertAuth) GetCACertificate() []byte {
	return ca.ca.GetCACertificate()
}
