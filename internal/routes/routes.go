package routes

import (
	"net/http"
	"os"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/api"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/hashlists"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/jobs"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/pkg/debug"
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

		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000" // fallback default
			debug.Warning("CORS_ALLOWED_ORIGIN not set, using default: %s", allowedOrigin)
			debug.Debug("Using fallback CORS origin: %s", allowedOrigin)
		}

		debug.Debug("Setting CORS headers with allowed origin: %s", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
func SetupRoutes(r *mux.Router) {
	debug.Info("Initializing route configuration")

	// Apply CORS middleware to all routes
	r.Use(CORSMiddleware)
	debug.Debug("Applied CORS middleware to all routes")

	// Configure public routes
	debug.Debug("Setting up public routes")
	r.HandleFunc("/api/login", auth.LoginHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/logout", auth.LogoutHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/check-auth", auth.CheckAuthHandler).Methods("GET", "OPTIONS")

	// Configure protected routes
	debug.Debug("Setting up protected routes")
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.JWTMiddleware)

	// Main feature routes
	debug.Debug("Configuring main feature routes")
	protected.HandleFunc("/dashboard", dashboard.GetDashboard).Methods("GET")
	protected.HandleFunc("/hashlists", hashlists.GetHashlists).Methods("GET")
	protected.HandleFunc("/jobs", jobs.GetJobs).Methods("GET")

	// API subrouter configuration
	debug.Debug("Setting up API subrouter")
	apiRouter := protected.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/some-endpoint", api.SomeAPIHandler).Methods("GET")

	// Agent subrouter configuration
	// debug.Debug("Setting up Agent subrouter")
	// agentRouter := protected.PathPrefix("/agent").Subrouter()
	// agentRouter.HandleFunc("/some-endpoint", agent.SomeAgentHandler).Methods("GET")

	debug.Info("Route configuration completed successfully")
}

// TODO: Implement agent-related functionality
// func unusedAgentPlaceholder() {
// 	_ = agent.SomeFunction // Replace SomeFunction with an actual function from the agent package
// }
