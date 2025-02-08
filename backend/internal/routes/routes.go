package routes

import (
	"bufio"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/auth"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/middleware"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
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

		// Get request origin
		origin := r.Header.Get("Origin")
		debug.Debug("Request origin: %s", origin)

		// Always allow the request origin if it's present
		// This is safe because we're using cookie-based auth
		if origin != "" {
			debug.Debug("Setting CORS origin to match request: %s", origin)
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			debug.Warning("No origin header present in request")
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

		// Log final headers for debugging
		debug.Debug("Final response headers: %v", w.Header())

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
	emailService := email.NewService(sqlDB)

	// Initialize configuration
	appConfig := config.NewConfig()
	debug.Info("Application configuration initialized")

	// Create API router with logging
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(loggingMiddleware)
	debug.Info("Created API router with logging middleware")

	// Setup TLS routes
	SetupTLSRoutes(r, tlsProvider)

	// Create MFA handler
	mfaHandler := auth.NewMFAHandler(database, emailService)

	// Setup public routes
	SetupPublicRoutes(apiRouter, database, agentService, appConfig, tlsProvider)

	// Setup JWT protected routes
	jwtRouter := apiRouter.PathPrefix("").Subrouter()
	jwtRouter.Use(middleware.RequireAuth(database))
	jwtRouter.Use(loggingMiddleware)

	// Setup feature-specific routes
	SetupDashboardRoutes(jwtRouter)
	SetupHashlistRoutes(jwtRouter)
	SetupJobRoutes(jwtRouter)
	SetupAgentRoutes(jwtRouter, agentService)
	SetupVoucherRoutes(jwtRouter, voucherService)
	SetupAdminRoutes(jwtRouter, database, emailService)
	SetupUserRoutes(jwtRouter, database)
	SetupMFARoutes(jwtRouter, mfaHandler, database, emailService)
	SetupWebSocketRoutes(r, agentService, tlsProvider)

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
