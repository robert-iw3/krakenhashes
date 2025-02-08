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
			debug.Debug("Checking authentication")

			// Skip middleware for OPTIONS requests
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from cookie
			cookie, err := r.Cookie("token")
			if err != nil {
				debug.Warning("No auth token found in cookies")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

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

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			r = r.WithContext(ctx)

			debug.Debug("Authentication successful for user: %s", userID)
			next.ServeHTTP(w, r)
		})
	}
}
