package middleware

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
)

// AdminOnly middleware ensures that only admin users can access the route
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Checking admin authorization")

		// Get token from cookie
		cookie, err := r.Cookie("token")
		if err != nil {
			debug.Warning("No auth token found in cookies")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token and get role
		role, err := jwt.GetUserRole(cookie.Value)
		if err != nil {
			debug.Warning("Invalid token: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is admin
		if role != "admin" {
			debug.Warning("Non-admin user attempted to access admin route (role: %s)", role)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		debug.Debug("Admin access granted for user with role: %s", role)
		next.ServeHTTP(w, r)
	})
}
