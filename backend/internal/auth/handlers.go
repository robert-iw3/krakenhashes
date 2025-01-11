package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/database"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the expected JSON structure for login attempts
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

/*
 * LoginHandler processes user login requests.
 * It validates credentials, generates a JWT token, and sets a secure cookie.
 *
 * Request body expects JSON:
 * {
 *   "username": "string",
 *   "password": "string"
 * }
 *
 * Responses:
 *   - 200: Successfully logged in, sets auth cookie
 *   - 400: Invalid request format
 *   - 401: Invalid credentials
 *   - 500: Server error (token generation/storage)
 */
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Processing login request")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Warning("Failed to decode login request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	debug.Debug("Login request decoded for user: %s", req.Username)

	user, err := database.GetUserByUsername(req.Username)
	if err != nil {
		debug.Info("Failed login attempt for user '%s': %v", req.Username, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate hash from provided password for comparison
	debug.Debug("Comparing password hashes for user '%s'", req.Username)
	debug.Debug("Stored hash in database: %s", user.PasswordHash)

	// Hash the provided password with the same cost factor
	hashedInput, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		debug.Error("Failed to hash input password for comparison: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	debug.Debug("Generated hash from input: %s", string(hashedInput))

	// Compare the hashes
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		debug.Info("Password hash comparison failed for user '%s'", req.Username)
		debug.Debug("Hash comparison error: %v", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	debug.Info("Password hash validated for user '%s'", req.Username)

	// Generate JWT token with string UUID
	token, err := jwt.GenerateToken(user.ID.String())
	if err != nil {
		debug.Error("Failed to generate token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store token in database
	if err := database.StoreToken(user.ID.String(), token); err != nil {
		debug.Error("Failed to store token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour * 24 * 7 / time.Second), // 1 week
	})
	debug.Info("User '%s' successfully logged in", req.Username)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

/*
 * LogoutHandler processes user logout requests.
 * It removes the token from the database and invalidates the auth cookie.
 *
 * Responses:
 *   - 200: Successfully logged out
 *   - 500: Error removing token from database
 */
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Processing logout request")

	cookie, err := r.Cookie("token")
	if err == nil {
		if err := database.RemoveToken(cookie.Value); err != nil {
			debug.Error("Failed to remove token from database: %v", err)
			http.Error(w, "Error removing token", http.StatusInternalServerError)
			return
		}
		debug.Debug("Token removed from database")
	} else {
		debug.Debug("No token cookie found during logout")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	debug.Info("User successfully logged out")

	w.WriteHeader(http.StatusOK)
}

/*
 * CheckAuthHandler verifies if the current request has valid authentication.
 * It checks for the presence of a valid JWT token in the cookies and verifies
 * it exists in the database.
 *
 * Responses:
 *   - 200: JSON response indicating authentication status
 *     {
 *       "authenticated": boolean
 *     }
 */
func CheckAuthHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Checking authentication status")

	cookie, err := r.Cookie("token")
	if err != nil {
		debug.Debug("No auth token found in cookies")
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}

	// Validate token cryptographically
	userID, err := jwt.ValidateJWT(cookie.Value)
	if err != nil {
		debug.Info("Invalid token found: %v", err)
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}

	// Verify token exists in database
	exists, err := database.TokenExists(cookie.Value)
	if err != nil {
		debug.Error("Error checking token in database: %v", err)
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}
	if !exists {
		debug.Warning("Token not found in database for user ID: %s", userID)
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}

	debug.Info("Valid authentication found for user ID: %s", userID)
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
}
