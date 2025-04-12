package middleware

import (
	"context"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
)

// RequireAuth middleware ensures that only authenticated users can access the route
func RequireAuth(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debug.Debug("Checking authentication for %s %s", r.Method, r.URL.Path)

			// Skip middleware for OPTIONS requests
			if r.Method == "OPTIONS" {
				debug.Debug("Skipping auth check for OPTIONS request")
				next.ServeHTTP(w, r)
				return
			}

			// Log all cookies for debugging
			debug.Debug("Request cookies: %v", r.Cookies())

			// Get token from cookie
			cookie, err := r.Cookie("token")
			if err != nil {
				debug.Warning("No auth token found in cookies for %s %s", r.Method, r.URL.Path)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			debug.Debug("Found auth token cookie: %s", cookie.Name)

			// Validate token and get user ID
			userID, err := jwt.ValidateJWT(cookie.Value)
			if err != nil {
				debug.Warning("Invalid token: %v", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify token exists in database
			exists, err := database.TokenExists(cookie.Value)
			if err != nil {
				debug.Error("Error checking token in database: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if !exists {
				debug.Warning("Token not found in database for user ID: %s", userID)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get user role from token
			role, err := jwt.GetUserRole(cookie.Value)
			if err != nil {
				// This shouldn't happen if ValidateJWT passed, but handle defensively
				debug.Warning("Failed to get role from valid token: %v", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID and role to request context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "user_role", role) // Add role to context
			r = r.WithContext(ctx)

			debug.Debug("Authentication successful for user: %s with role: %s", userID, role)
			next.ServeHTTP(w, r)
		})
	}
}
