package routes

import (
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/auth"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/middleware"
	"github.com/gorilla/mux"
)

// SetupMFARoutes sets up the MFA-related routes
func SetupMFARoutes(router *mux.Router, mfaHandler *auth.MFAHandler, database *db.DB, emailService *email.Service) {
	// Protected routes (require authentication)
	protected := router.PathPrefix("/user/mfa").Subrouter()
	protected.Use(middleware.RequireAuth(database))

	// Get user MFA settings
	protected.HandleFunc("/settings", mfaHandler.GetUserMFASettings).Methods("GET", "OPTIONS")

	// Enable MFA
	protected.HandleFunc("/enable", mfaHandler.EnableMFA).Methods("POST", "OPTIONS")

	// Verify MFA setup
	protected.HandleFunc("/verify-setup", mfaHandler.VerifyMFASetup).Methods("POST", "OPTIONS")

	// Disable MFA
	protected.HandleFunc("/disable", mfaHandler.DisableMFA).Methods("POST", "OPTIONS")

	// Generate backup codes
	protected.HandleFunc("/backup-codes", mfaHandler.GenerateBackupCodes).Methods("POST", "OPTIONS")

	// Send email MFA code
	protected.HandleFunc("/email/send", mfaHandler.SendEmailMFACode).Methods("POST", "OPTIONS")

	// Verify MFA code
	protected.HandleFunc("/verify", mfaHandler.VerifyMFACode).Methods("POST", "OPTIONS")

	// Update preferred MFA method
	protected.HandleFunc("/preferred-method", mfaHandler.UpdatePreferredMFAMethod).Methods("PUT", "OPTIONS")
}
