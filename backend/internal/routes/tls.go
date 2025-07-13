package routes

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/agent"
	tlshandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupTLSRoutes configures TLS-related routes
func SetupTLSRoutes(r *mux.Router, tlsProvider tls.Provider, database *db.DB) {
	// Initialize TLS handler
	tlsHandler := tlshandler.NewHandler(tlsProvider)
	debug.Info("TLS handler initialized")

	// Initialize agent repository for certificate renewal
	agentRepo := repository.NewAgentRepository(database)
	
	// Initialize certificate renewal handler
	certRenewalHandler := agent.NewCertificateRenewalHandler(tlsProvider, agentRepo)
	debug.Info("Certificate renewal handler initialized")

	// Create HTTP router for CA certificate and certificate renewal (no TLS)
	httpRouter := mux.NewRouter()
	httpRouter.Use(CORSMiddleware)
	httpRouter.HandleFunc("/ca.crt", tlsHandler.ServeCACertificate).Methods("GET", "HEAD", "OPTIONS")
	httpRouter.HandleFunc("/api/agent/renew-certificates", certRenewalHandler.HandleCertificateRenewal).Methods("POST", "OPTIONS")
	debug.Info("Created HTTP router for CA certificate and certificate renewal")

	// Start HTTP server for CA certificate and certificate renewal
	go func() {
		debug.Info("Starting HTTP server for CA certificate and certificate renewal on port 1337")
		if err := http.ListenAndServe(":1337", httpRouter); err != nil {
			debug.Error("HTTP server failed: %v", err)
		}
	}()
}
