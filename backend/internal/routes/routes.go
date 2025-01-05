package routes

import (
	"bufio"
	cryptotls "crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom-backend/internal/config"
	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/agent"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/hashlists"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/jobs"
	tlshandler "github.com/ZerkerEOD/hashdom-backend/internal/handlers/tls"
	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/vouchers"
	wshandler "github.com/ZerkerEOD/hashdom-backend/internal/handlers/websocket"
	"github.com/ZerkerEOD/hashdom-backend/internal/repository"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	wsservice "github.com/ZerkerEOD/hashdom-backend/internal/services/websocket"
	"github.com/ZerkerEOD/hashdom-backend/internal/tls"
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
		debug.Info("Processing CORS for %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Get allowed origin from environment
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "https://localhost:3000" // fallback default
			debug.Warning("CORS_ALLOWED_ORIGIN not set, using default: %s", allowedOrigin)
		}

		// Get request origin
		origin := r.Header.Get("Origin")
		debug.Debug("Request origin: %s, Allowed origin: %s", origin, allowedOrigin)

		// Always use the actual origin if it matches our expected host
		if origin != "" && (strings.HasPrefix(origin, "https://localhost:") || strings.HasPrefix(origin, "http://localhost:")) {
			debug.Debug("Using request origin: %s", origin)
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			debug.Warning("Using default allowed origin: %s", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		// Set standard CORS headers
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, X-Agent-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			debug.Info("Handling OPTIONS preflight request from origin: %s", origin)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log headers for debugging
		debug.Debug("Response headers: %v", w.Header())

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
 *   - API Key authentication (agent routes)
 */
func SetupRoutes(r *mux.Router, sqlDB *sql.DB, tlsProvider tls.Provider) {
	debug.Info("Initializing route configuration")

	// Create our custom DB wrapper
	database := &db.DB{DB: sqlDB}
	debug.Debug("Created custom DB wrapper")

	// Apply CORS middleware at the root level
	r.Use(CORSMiddleware)
	debug.Info("Applied CORS middleware to root router")

	// Initialize repositories and services
	debug.Debug("Initializing repositories and services")
	agentRepo := repository.NewAgentRepository(database)
	voucherRepo := repository.NewClaimVoucherRepository(database)
	agentService := services.NewAgentService(agentRepo, voucherRepo)
	voucherService := services.NewClaimVoucherService(voucherRepo)

	// Initialize TLS handler
	tlsHandler := tlshandler.NewHandler(tlsProvider)
	debug.Info("TLS handler initialized")

	// Initialize configuration
	appConfig := config.NewConfig()
	debug.Info("Application configuration initialized")

	// Create API router with logging
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(loggingMiddleware)
	debug.Info("Created API router with logging middleware")

	// 1. Public routes (no authentication required)
	debug.Debug("Setting up public routes")
	// Auth endpoints
	apiRouter.HandleFunc("/login", auth.LoginHandler).Methods("POST", "OPTIONS")
	apiRouter.HandleFunc("/logout", auth.LogoutHandler).Methods("POST", "OPTIONS")
	apiRouter.HandleFunc("/check-auth", auth.CheckAuthHandler).Methods("GET", "OPTIONS")
	debug.Info("Configured authentication endpoints: /login, /logout, /check-auth")

	// CA certificate endpoint - publicly accessible over HTTP
	publicRouter := r.PathPrefix("").Subrouter()
	publicRouter.Use(CORSMiddleware)
	publicRouter.HandleFunc("/ca.crt", tlsHandler.ServeCACertificate).Methods("GET", "HEAD", "OPTIONS")
	debug.Info("Configured public CA certificate endpoint: /ca.crt")

	// Health check endpoint - publicly accessible
	publicRouter.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		debug.Info("Health check request from %s", r.RemoteAddr)
		debug.Debug("Health check request headers: %v", r.Header)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET", "OPTIONS")
	debug.Info("Configured health check endpoint: /api/health")

	// Agent registration endpoint
	registrationHandler := handlers.NewRegistrationHandler(agentService, appConfig, tlsProvider)
	apiRouter.HandleFunc("/agent/register", registrationHandler.HandleRegistration).Methods("POST", "OPTIONS")
	debug.Info("Configured agent registration endpoint: /agent/register")

	// 2. JWT Protected routes (frontend access)
	debug.Debug("Setting up JWT protected routes")
	jwtRouter := apiRouter.PathPrefix("").Subrouter()
	jwtRouter.Use(auth.JWTMiddleware)
	jwtRouter.Use(loggingMiddleware)
	debug.Info("Applied JWT middleware to protected routes")

	// Client certificate endpoint - protected by JWT
	jwtRouter.HandleFunc("/client-cert", tlsHandler.ServeClientCertificate).Methods("GET", "OPTIONS")
	debug.Info("Configured protected client certificate endpoint: /api/client-cert")

	// Dashboard routes
	jwtRouter.HandleFunc("/dashboard", dashboard.GetDashboard).Methods("GET", "OPTIONS")
	debug.Info("Configured dashboard endpoint: /dashboard")

	// Hashlist routes
	jwtRouter.HandleFunc("/hashlists", hashlists.GetHashlists).Methods("GET", "OPTIONS")
	debug.Info("Configured hashlists endpoint: /hashlists")

	// Job routes
	jwtRouter.HandleFunc("/jobs", jobs.GetJobs).Methods("GET", "OPTIONS")
	debug.Info("Configured jobs endpoint: /jobs")

	// Agent management routes
	agentHandler := agent.NewAgentHandler(agentService)
	jwtRouter.HandleFunc("/agents", agentHandler.ListAgents).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}", agentHandler.GetAgent).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}", agentHandler.DeleteAgent).Methods("DELETE", "OPTIONS")
	debug.Info("Configured agent management endpoints: /agents")

	// Voucher management routes
	voucherHandler := vouchers.NewVoucherHandler(voucherService)
	jwtRouter.HandleFunc("/vouchers/temp", voucherHandler.GenerateVoucher).Methods("POST", "OPTIONS")
	jwtRouter.HandleFunc("/vouchers", voucherHandler.ListVouchers).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/vouchers/{code}/disable", voucherHandler.DeactivateVoucher).Methods("DELETE", "OPTIONS")
	debug.Info("Configured voucher management endpoints: /vouchers")

	// 3. API Key Protected routes (agent communication)
	debug.Debug("Setting up API key protected routes")
	wsService := wsservice.NewService(agentService)

	// Get TLS configuration for WebSocket handler
	tlsConfig, err := tlsProvider.GetTLSConfig()
	if err != nil {
		debug.Error("Failed to get TLS configuration: %v", err)
		return
	}

	wsRouter := r.PathPrefix("/ws").Subrouter()
	wsRouter.Use(auth.APIKeyMiddleware(agentService))
	wsRouter.Use(loggingMiddleware)
	// Create a copy of the TLS config with required client certs for agent connections
	agentTLSConfig := *tlsConfig // Make a copy
	agentTLSConfig.ClientAuth = cryptotls.RequireAndVerifyClientCert
	wsHandler := wshandler.NewHandler(wsService, agentService, &agentTLSConfig)
	wsRouter.HandleFunc("/agent", wsHandler.ServeWS)
	debug.Info("Configured WebSocket endpoint: /ws/agent with TLS: %v", tlsConfig != nil)
	if tlsConfig != nil {
		debug.Debug("WebSocket TLS Configuration:")
		debug.Debug("- Client Auth: %v", agentTLSConfig.ClientAuth)
		debug.Debug("- Client CAs: %v", agentTLSConfig.ClientCAs != nil)
		debug.Debug("- Certificates: %d", len(agentTLSConfig.Certificates))
	}

	debug.Info("Route configuration completed successfully")
	logRegisteredRoutes(r)
}

// loggingMiddleware logs details about each request
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		debug.Info("Request received: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		debug.Debug("Request headers: %v", r.Header)

		// Create a response wrapper to capture the status code
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		debug.Info("Request completed: %s %s - Status: %d - Duration: %v",
			r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements the http.Hijacker interface to support WebSocket connections
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// logRegisteredRoutes prints all registered routes for debugging
func logRegisteredRoutes(r *mux.Router) {
	debug.Info("Registered routes:")
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			pathTemplate = "<unknown>"
		}
		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"ANY"}
		}
		debug.Info("Route: %s [%s]", pathTemplate, strings.Join(methods, ", "))
		return nil
	})
}

// TODO: Implement agent-related functionality
// func unusedAgentPlaceholder() {
// 	_ = agent.SomeFunction // Replace SomeFunction with an actual function from the agent package
// }
