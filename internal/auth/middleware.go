package auth

import (
	"context"
	"net/http"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/pkg/debug"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/pkg/jwt"
)

/*
 * JWTMiddleware handles JWT-based authentication for protected routes.
 * It validates the JWT token from cookies and adds the user ID to the request context.
 *
 * Authentication Flow:
 * 1. Extracts JWT token from 'token' cookie
 * 2. Validates the token
 * 3. Adds user ID to request context if valid
 *
 * Parameters:
 *   - next: The next handler in the middleware chain
 *
 * Returns:
 *   - http.Handler: Middleware handler that processes JWT authentication
 *
 * Context Values Added:
 *   - "user_id": (int) The authenticated user's ID
 *
 * Response Codes:
 *   - 401: Unauthorized - Missing or invalid token
 *   - Next handler's response if authentication succeeds
 */
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Processing JWT middleware for request: %s %s from %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
		)

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

		// Validate the token
		claims, err := jwt.ValidateToken(cookie.Value)
		if err != nil {
			debug.Error("Token validation failed for request from %s: %v",
				r.RemoteAddr,
				err,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Info("Token validated successfully for user ID: %d, IP: %s",
			claims.UserID,
			r.RemoteAddr,
		)

		// Add user ID to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		debug.Debug("Added user ID %d to request context", claims.UserID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
		debug.Debug("Completed processing request for user ID: %d, Path: %s",
			claims.UserID,
			r.URL.Path,
		)
	})
}
