package tls

import (
	"fmt"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// Handler manages TLS-related HTTP endpoints
type Handler struct {
	provider tls.Provider
}

// NewHandler creates a new TLS handler
func NewHandler(provider tls.Provider) *Handler {
	return &Handler{
		provider: provider,
	}
}

// ServeCACertificate serves the CA certificate for browser installation
func (h *Handler) ServeCACertificate(w http.ResponseWriter, r *http.Request) {
	debug.Info("Serving CA certificate to client: %s", r.RemoteAddr)

	// Get CA certificate
	cert, err := h.provider.ExportCACertificate()
	if err != nil {
		debug.Error("Failed to export CA certificate: %v", err)
		http.Error(w, "Failed to get CA certificate", http.StatusInternalServerError)
		return
	}

	// Set headers for browser download
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", "attachment; filename=krakenhashes-ca.crt")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cert)))

	// Write certificate
	if _, err := w.Write(cert); err != nil {
		debug.Error("Failed to write CA certificate to response: %v", err)
		return
	}

	debug.Info("Successfully served CA certificate")
}

// ServeClientCertificate serves the client certificate and private key
func (h *Handler) ServeClientCertificate(w http.ResponseWriter, r *http.Request) {
	debug.Info("Serving client certificate to client: %s", r.RemoteAddr)

	// Get client certificate and key
	certPEM, keyPEM, err := h.provider.GetClientCertificate()
	if err != nil {
		debug.Error("Failed to get client certificate: %v", err)
		http.Error(w, "Failed to get client certificate", http.StatusInternalServerError)
		return
	}

	// Set headers for browser download
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=krakenhashes-client.pem")

	// Write certificate and key
	if _, err := w.Write(certPEM); err != nil {
		debug.Error("Failed to write client certificate to response: %v", err)
		return
	}
	if _, err := w.Write(keyPEM); err != nil {
		debug.Error("Failed to write client key to response: %v", err)
		return
	}

	debug.Info("Successfully served client certificate")
}
