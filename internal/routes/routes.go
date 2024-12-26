package routes

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/ZerkerEOD/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/agent"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/api"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/hashlists"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/jobs"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/vouchers"
	wshandler "github.com/ZerkerEOD/hashdom-backend/internal/handlers/websocket"
	"github.com/ZerkerEOD/hashdom-backend/internal/repository"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	wsservice "github.com/ZerkerEOD/hashdom-backend/internal/services/websocket"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/mux"
)

/*
 * Package routes handles the setup and configuration of all application routes.
 * It includes middleware for CORS and authentication, and organizes routes into
 * logical groups for different parts of the application.
 */

/*
 * CORSMiddleware handles CORS headers for all requests.
 * It configures cross-origin resource sharing based on environment settings.
 *
 * Configuration:
 *   - Uses CORS_ALLOWED_ORIGIN environment variable
 *   - Falls back to http://localhost:3000 if not set
 *
 * Headers Set:
 *   - Access-Control-Allow-Origin
 *   - Access-Control-Allow-Methods
 *   - Access-Control-Allow-Headers
 *   - Access-Control-Allow-Credentials
 *
 * Parameters:
 *   - next: The next handler in the middleware chain
 *
 * Returns:
 *   - http.Handler: Middleware handler that processes CORS
 */
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Processing CORS middleware for request: %s %s", r.Method, r.URL.Path)

		// Special handling for WebSocket connections
		if r.Header.Get("Upgrade") == "websocket" {
			debug.Debug("WebSocket connection detected, bypassing CORS restrictions")
			next.ServeHTTP(w, r)
			return
		}

		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000" // fallback default
			debug.Warning("CORS_ALLOWED_ORIGIN not set, using default: %s", allowedOrigin)
			debug.Debug("Using fallback CORS origin: %s", allowedOrigin)
		}

		debug.Debug("Setting CORS headers with allowed origin: %s", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Claim-Code, X-Download-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			debug.Debug("Handling OPTIONS preflight request from origin: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusOK)
			return
		}

		debug.Debug("CORS headers set, proceeding with request")
		next.ServeHTTP(w, r)
	})
}

/*
 * SetupRoutes configures all application routes and middleware.
 *
 * Route Groups:
 *   - Public Routes (/api/login, /api/logout, /api/check-auth)
 *   - Protected Routes (requires authentication)
 *     - Dashboard (/api/dashboard)
 *     - Hashlists (/api/hashlists)
 *     - Jobs (/api/jobs)
 *     - API endpoints (/api/api/...)
 *     - Agent endpoints (/api/agent/...)
 *
 * Middleware Applied:
 *   - CORS middleware (all routes)
 *   - JWT authentication (protected routes)
 *
 * Parameters:
 *   - r: The root router to configure
 */
func SetupRoutes(r *mux.Router, sqlDB *sql.DB) {
	debug.Info("Initializing route configuration")

	// Create our custom DB wrapper
	database := &db.DB{DB: sqlDB}
	debug.Debug("Created custom DB wrapper")

	// Apply CORS middleware to all routes
	r.Use(CORSMiddleware)
	debug.Debug("Applied CORS middleware to all routes")

	// Configure public routes
	debug.Debug("Setting up public routes")
	r.HandleFunc("/api/login", auth.LoginHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/logout", auth.LogoutHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/check-auth", auth.CheckAuthHandler).Methods("GET", "OPTIONS")

	// Agent routes
	debug.Debug("Setting up agent routes")
	agentRepo := repository.NewAgentRepository(database)
	voucherRepo := repository.NewClaimVoucherRepository(database)
	agentService := services.NewAgentService(agentRepo, voucherRepo)
	agentHandler := agent.NewAgentHandler(agentService)

	// Initialize CA manager for certificate operations
	caCertPath := os.Getenv("CA_CERT_PATH")
	if caCertPath == "" {
		debug.Error("CA_CERT_PATH environment variable is not set")
		return
	}

	caKeyPath := os.Getenv("CA_KEY_PATH")
	if caKeyPath == "" {
		debug.Error("CA_KEY_PATH environment variable is not set")
		return
	}

	caManager, err := auth.NewCAManager(caCertPath, caKeyPath)
	if err != nil {
		debug.Error("Failed to initialize CA manager: %v", err)
		return
	}

	// Create registration handler
	registrationHandler := handlers.NewRegistrationHandler(agentService, caManager)

	// Agent registration endpoints (unprotected)
	r.HandleFunc("/api/agent/register", registrationHandler.HandleRegistration).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/agent/cert", registrationHandler.HandleCertificateDownload).Methods("GET", "OPTIONS")

	// Configure protected routes
	debug.Debug("Setting up protected routes")
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.JWTMiddleware)

	// Main feature routes
	debug.Debug("Configuring main feature routes")
	protected.HandleFunc("/dashboard", dashboard.GetDashboard).Methods("GET")
	protected.HandleFunc("/hashlists", hashlists.GetHashlists).Methods("GET")
	protected.HandleFunc("/jobs", jobs.GetJobs).Methods("GET")

	protected.HandleFunc("/agents", agentHandler.ListAgents).Methods("GET", "OPTIONS")
	protected.HandleFunc("/agents/{id}", agentHandler.GetAgent).Methods("GET", "OPTIONS")
	protected.HandleFunc("/agents/{id}", agentHandler.DeleteAgent).Methods("DELETE", "OPTIONS")

	// WebSocket route for agent connections (requires certificate auth)
	debug.Debug("Setting up WebSocket route for agent connections")
	wsService := wsservice.NewService(agentService)
	wsHandler := wshandler.NewHandler(wsService)

	// Add certificate authentication middleware for WebSocket connections
	wsRouter := r.PathPrefix("/ws").Subrouter()
	wsRouter.Use(auth.CertificateAuthMiddleware)
	wsRouter.HandleFunc("/agent", wsHandler.ServeWS)

	// Voucher routes
	debug.Debug("Setting up voucher routes")
	voucherService := services.NewClaimVoucherService(voucherRepo)
	voucherHandler := vouchers.NewVoucherHandler(voucherService)

	protected.HandleFunc("/vouchers/temp", voucherHandler.GenerateVoucher).Methods("POST", "OPTIONS")
	protected.HandleFunc("/vouchers", voucherHandler.ListVouchers).Methods("GET", "OPTIONS")
	protected.HandleFunc("/vouchers/{code}/disable", voucherHandler.DeactivateVoucher).Methods("DELETE", "OPTIONS")

	// API subrouter configuration
	debug.Debug("Setting up API subrouter")
	apiRouter := protected.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/some-endpoint", api.SomeAPIHandler).Methods("GET")

	debug.Info("Route configuration completed successfully")
}

// TODO: Implement agent-related functionality
// func unusedAgentPlaceholder() {
// 	_ = agent.SomeFunction // Replace SomeFunction with an actual function from the agent package
// }
