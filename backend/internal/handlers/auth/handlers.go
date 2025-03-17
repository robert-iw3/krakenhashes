package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the expected JSON structure for login attempts
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Helper function to get cookie domain from request host
func getCookieDomain(host string) string {
	debug.Debug("Getting cookie domain from host: %s", host)

	// Always strip port number since frontend and backend are on different ports
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// For development environments (localhost/127.0.0.1), don't set domain
	if host == "localhost" || host == "127.0.0.1" {
		debug.Debug("Development environment detected, not setting cookie domain")
		return ""
	}

	debug.Debug("Using cookie domain: %s", host)
	return host
}

// Helper function to set auth cookie
func setAuthCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	debug.Debug("Setting auth cookie - MaxAge: %d", maxAge)

	cookie := &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,                 // Require HTTPS
		SameSite: http.SameSiteLaxMode, // Allow cross-site (needed for different ports)
		Path:     "/",
		MaxAge:   maxAge,
	}

	// Get domain (will be empty for localhost/127.0.0.1)
	domain := getCookieDomain(r.Host)
	if domain != "" {
		cookie.Domain = domain
		debug.Debug("Setting cookie domain: %s", domain)
	} else {
		debug.Debug("No domain set for cookie (development environment)")
	}

	http.SetCookie(w, cookie)
	debug.Debug("Auth cookie set with attributes: domain=%s, secure=true, sameSite=lax, httpOnly=true, path=/",
		cookie.Domain)
}

// generateAuthToken creates a new JWT token for the user
func (h *Handler) generateAuthToken(user *models.User) (string, error) {
	return jwt.GenerateToken(user.ID.String(), user.Role)
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
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Processing login request")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Warning("Failed to decode login request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	debug.Debug("Login request decoded for user: %s", req.Username)

	// Prevent login with system user
	if req.Username == "system" {
		debug.Warning("Attempted login with system user account")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		debug.Info("Failed login attempt for user '%s': %v", req.Username, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Prevent login with system user by ID
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" || user.Role == "system" {
		debug.Warning("Attempted login with system user account")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		debug.Info("Invalid password for user '%s'", req.Username)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check user's MFA settings and global MFA requirement
	mfaSettings, err := h.db.GetUserMFASettings(user.ID.String())
	if err != nil {
		debug.Error("error checking user MFA settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if MFA is globally required
	globalMFARequired, err := h.db.IsMFARequired()
	if err != nil {
		debug.Error("error checking global MFA requirement: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If either user has MFA enabled or it's globally required
	if mfaSettings.MFAEnabled || globalMFARequired {
		// Create MFA session
		sessionToken := uuid.New().String()
		session, err := h.db.CreateMFASession(user.ID.String(), sessionToken)
		if err != nil {
			debug.Error("error creating MFA session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If email is the preferred method and it's available, send the code
		if mfaSettings.PreferredMFAMethod == "email" && contains(mfaSettings.MFAType, "email") {
			code, err := generateEmailCode()
			if err != nil {
				debug.Error("error generating email code: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			err = h.db.StoreEmailMFACode(user.ID.String(), code)
			if err != nil {
				debug.Error("error storing email code: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Send email synchronously
			err = h.emailService.SendMFACode(r.Context(), user.Email, code)
			if err != nil {
				debug.Error("error sending MFA email: %v", err)
				http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
				return
			}
		}

		// Return MFA required response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mfa_required":     true,
			"session_token":    sessionToken,
			"mfa_type":         mfaSettings.MFAType,
			"preferred_method": mfaSettings.PreferredMFAMethod,
			"expires_at":       session.ExpiresAt.Format(time.RFC3339),
		})
		return
	}

	// If no MFA required, proceed with normal login
	token, err := h.generateAuthToken(user)
	if err != nil {
		debug.Error("Failed to generate token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.db.StoreToken(user.ID.String(), token); err != nil {
		debug.Error("Failed to store token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, r, token, int(time.Hour*24*7/time.Second)) // 1 week
	debug.Info("User '%s' successfully logged in", req.Username)

	json.NewEncoder(w).Encode(models.LoginResponse{
		Success: true,
		Token:   token,
	})
}

/*
 * LogoutHandler processes user logout requests.
 * It removes the token from the database and invalidates the auth cookie.
 *
 * Responses:
 *   - 200: Successfully logged out
 *   - 500: Error removing token from database
 */
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Processing logout request")

	cookie, err := r.Cookie("token")
	if err == nil {
		debug.Debug("Found token cookie, removing from database")
		if err := h.db.RemoveToken(cookie.Value); err != nil {
			debug.Error("Failed to remove token from database: %v", err)
			http.Error(w, "Error removing token", http.StatusInternalServerError)
			return
		}
		debug.Debug("Token removed from database successfully")
	} else {
		debug.Debug("No token cookie found during logout: %v", err)
	}

	setAuthCookie(w, r, "", -1) // Expire the cookie
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
func (h *Handler) CheckAuthHandler(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Checking authentication status")

	cookie, err := r.Cookie("token")
	if err != nil {
		debug.Debug("No auth token found in cookies")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"role":          nil,
		})
		return
	}

	// Validate token cryptographically
	userID, err := jwt.ValidateJWT(cookie.Value)
	if err != nil {
		debug.Info("Invalid token: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"role":          nil,
		})
		return
	}

	// Verify token exists in database
	exists, err := h.db.TokenExists(cookie.Value)
	if err != nil {
		debug.Error("Error checking token in database: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"role":          nil,
		})
		return
	}
	if !exists {
		debug.Warning("Token not found in database for user ID: %s", userID)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"role":          nil,
		})
		return
	}

	// Get user's role from token
	role, err := jwt.GetUserRole(cookie.Value)
	if err != nil {
		debug.Error("Error getting user role: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"role":          nil,
		})
		return
	}

	debug.Info("Valid authentication found for user ID: %s with role: %s", userID, role)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"role":          role,
	})
}

// Helper function to check if a string is in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
