package auth

import (
	"context"
	"net/http"

	"github.com/ZerkerEOD/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/ZerkerEOD/hashdom-backend/pkg/jwt"
)

/*
 * JWTMiddleware handles JWT-based authentication for protected routes.
 * It validates the JWT token from cookies and adds the user ID to the request context.
 *
 * Authentication Flow:
 * 1. Extracts JWT token from 'token' cookie
 * 2. Validates the token cryptographically
 * 3. Verifies the token exists in the database
 * 4. Adds user ID to request context if valid
 *
 * Parameters:
 *   - next: The next handler in the middleware chain
 *
 * Returns:
 *   - http.Handler: Middleware handler that processes JWT authentication
 *
 * Context Values Added:
 *   - "user_id": (string) The authenticated user's ID
 *
 * Response Codes:
 *   - 401: Unauthorized - Missing, invalid, or revoked token
 *   - Next handler's response if authentication succeeds
 */
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Processing JWT middleware for request: %s %s from %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
		)

		// Allow OPTIONS requests to pass through for CORS preflight
		if r.Method == "OPTIONS" {
			debug.Debug("Allowing OPTIONS request to pass through JWT middleware")
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from cookie
		cookie, err := r.Cookie("token")
		if err != nil {
			debug.Warning("No token cookie found in request from %s: %v",
				r.RemoteAddr,
				err,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Debug("Token cookie found in request")

		// Validate the token cryptographically
		userID, err := jwt.ValidateJWT(cookie.Value)
		if err != nil {
			debug.Error("Token validation failed for request from %s: %v",
				r.RemoteAddr,
				err,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Debug("Token cryptographically validated for user ID: %s", userID)

		// Verify token exists in database
		exists, err := database.TokenExists(cookie.Value)
		if err != nil {
			debug.Error("Error checking token in database: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !exists {
			debug.Warning("Token not found in database for user ID: %s", userID)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Info("Token validated successfully for user ID: %s, IP: %s",
			userID,
			r.RemoteAddr,
		)

		// Add user ID to context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		debug.Debug("Added user ID %s to request context", userID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
		debug.Debug("Completed processing request for user ID: %s, Path: %s",
			userID,
			r.URL.Path,
		)
	})
}

// NewClaimCodeMiddleware creates a middleware that validates claim codes
func NewClaimCodeMiddleware(voucherService *services.ClaimVoucherService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debug.Info("Processing claim code authentication")

			// Extract claim code from header
			claimCode := r.Header.Get("X-Claim-Code")
			if claimCode == "" {
				debug.Error("No claim code provided")
				http.Error(w, "Claim code required", http.StatusUnauthorized)
				return
			}

			// Validate claim code
			if err := voucherService.ValidateClaimCode(r.Context(), claimCode); err != nil {
				debug.Error("Invalid claim code: %v", err)
				http.Error(w, "Invalid claim code", http.StatusUnauthorized)
				return
			}

			debug.Info("Valid claim code provided")
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyMiddleware authenticates requests using API keys
func APIKeyMiddleware(agentService *services.AgentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debug.Info("Processing API key authentication for %s %s", r.Method, r.URL.Path)

			// Get API key from header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				debug.Error("No API key provided")
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}

			// Get agent ID from header
			agentID := r.Header.Get("X-Agent-ID")
			if agentID == "" {
				debug.Error("No agent ID provided")
				http.Error(w, "Agent ID required", http.StatusUnauthorized)
				return
			}

			// Validate API key and get agent
			agent, err := agentService.GetByAPIKey(r.Context(), apiKey)
			if err != nil {
				debug.Error("Invalid API key: %v", err)
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Store agent in context
			ctx := context.WithValue(r.Context(), "agent", agent)
			r = r.WithContext(ctx)

			debug.Info("API key authentication successful for agent %d", agent.ID)
			next.ServeHTTP(w, r)
		})
	}
}
