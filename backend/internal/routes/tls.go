package routes

import (
	"net/http"

	tlshandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupTLSRoutes configures TLS-related routes
func SetupTLSRoutes(r *mux.Router, tlsProvider tls.Provider) {
	// Initialize TLS handler
	tlsHandler := tlshandler.NewHandler(tlsProvider)
	debug.Info("TLS handler initialized")

	// Create HTTP router for CA certificate (no TLS)
	httpRouter := mux.NewRouter()
	httpRouter.Use(CORSMiddleware)
	httpRouter.HandleFunc("/ca.crt", tlsHandler.ServeCACertificate).Methods("GET", "HEAD", "OPTIONS")
	debug.Info("Created HTTP router for CA certificate")

	// Start HTTP server for CA certificate
	go func() {
		debug.Info("Starting HTTP server for CA certificate on port 1337")
		if err := http.ListenAndServe(":1337", httpRouter); err != nil {
			debug.Error("HTTP server failed: %v", err)
		}
	}()
}
