package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/password"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// SetupUserRoutes configures all user-related routes
func SetupUserRoutes(router *mux.Router, database *db.DB) {
	debug.Info("Setting up user routes")

	// Get user profile
	router.HandleFunc("/user/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from context (set by RequireAuth middleware)
		userID := r.Context().Value("user_id").(string)
		if userID == "" {
			debug.Warning("No user ID found in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Query user profile
		var profile struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
		}

		err := database.QueryRow(
			"SELECT id, username, email FROM users WHERE id = $1",
			userID,
		).Scan(&profile.ID, &profile.Username, &profile.Email)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			debug.Error("Failed to fetch user profile: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}).Methods("GET", "OPTIONS")

	// Update user profile
	router.HandleFunc("/user/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from context
		userID := r.Context().Value("user_id").(string)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse request body
		var update struct {
			Email           string `json:"email"`
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}

		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := database.Begin()
		if err != nil {
			debug.Error("Failed to start transaction: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Get current user data for verification
		var currentUser struct {
			PasswordHash string
			Email        string
		}
		err = tx.QueryRow(
			"SELECT password_hash, email FROM users WHERE id = $1",
			userID,
		).Scan(&currentUser.PasswordHash, &currentUser.Email)
		if err != nil {
			debug.Error("Failed to get current user data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Handle password change if requested
		if update.NewPassword != "" {
			// Require current password for password changes
			if update.CurrentPassword == "" {
				http.Error(w, "Current password is required to change password", http.StatusBadRequest)
				return
			}

			// Verify current password
			if err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(update.CurrentPassword)); err != nil {
				debug.Info("Invalid current password provided for user %s", userID)
				http.Error(w, "Invalid current password", http.StatusBadRequest)
				return
			}

			// Get auth settings for password validation
			settings, err := database.GetAuthSettings()
			if err != nil {
				debug.Error("Failed to get auth settings: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Validate new password against policy
			if err := password.Validate(update.NewPassword, settings); err != nil {
				debug.Info("Password validation failed: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Hash new password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(update.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				debug.Error("Failed to hash new password: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Update password
			_, err = tx.Exec(
				"UPDATE users SET password_hash = $1, last_password_change = CURRENT_TIMESTAMP WHERE id = $2",
				string(hashedPassword), userID,
			)
			if err != nil {
				debug.Error("Failed to update password: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Get user role for token generation
			var userRole string
			err = tx.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&userRole)
			if err != nil {
				debug.Error("Failed to get user role: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Generate and store new token
			token, err := jwt.GenerateToken(userID, userRole)
			if err != nil {
				debug.Error("Failed to generate new token: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if err := database.StoreToken(userID, token); err != nil {
				debug.Error("Failed to store new token: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Set new auth cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(time.Hour * 24 * 7 / time.Second), // 1 week
			})
		}

		// Update email if provided and changed
		if update.Email != "" && update.Email != currentUser.Email {
			// Check if email is already in use
			var exists bool
			err = tx.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id != $2)",
				update.Email, userID,
			).Scan(&exists)
			if err != nil {
				debug.Error("Failed to check email existence: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if exists {
				http.Error(w, "Email already in use", http.StatusConflict)
				return
			}

			// Update email
			_, err = tx.Exec(
				"UPDATE users SET email = $1 WHERE id = $2",
				update.Email, userID,
			)
			if err != nil {
				debug.Error("Failed to update email: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			debug.Error("Failed to commit transaction: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Profile updated successfully",
		})
	}).Methods("PUT", "OPTIONS")

	debug.Info("User routes setup complete")
}
