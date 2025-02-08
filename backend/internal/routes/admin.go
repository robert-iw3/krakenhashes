package routes

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin/auth"
	emailhandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/middleware"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupAdminRoutes configures all admin-related routes
func SetupAdminRoutes(router *mux.Router, database *db.DB, emailService *email.Service) *mux.Router {
	debug.Debug("Setting up admin routes")

	// Create handlers
	authSettingsHandler := auth.NewAuthSettingsHandler(database)
	emailHandler := emailhandler.NewHandler(emailService)

	// Create admin router
	adminRouter := router.PathPrefix("/admin").Subrouter()

	// Apply admin middleware
	adminRouter.Use(middleware.AdminOnly)

	// Auth settings routes
	adminRouter.HandleFunc("/auth/settings", authSettingsHandler.GetSettings).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/auth/settings", authSettingsHandler.UpdateSettings).Methods(http.MethodPut, http.MethodOptions)
	adminRouter.HandleFunc("/auth/settings/mfa", authSettingsHandler.GetMFASettings).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/auth/settings/mfa", authSettingsHandler.UpdateMFASettings).Methods(http.MethodPut, http.MethodOptions)
	adminRouter.HandleFunc("/auth/settings/password", authSettingsHandler.GetPasswordPolicy).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/auth/settings/security", authSettingsHandler.GetAccountSecurity).Methods(http.MethodGet, http.MethodOptions)

	// Email configuration endpoints
	adminRouter.HandleFunc("/email/config", emailHandler.GetConfig).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/email/config", emailHandler.UpdateConfig).Methods("POST", "PUT", "OPTIONS")
	adminRouter.HandleFunc("/email/test", emailHandler.TestConfig).Methods("POST", "OPTIONS")

	// Email template endpoints
	adminRouter.HandleFunc("/email/templates", emailHandler.ListTemplates).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/email/templates", emailHandler.CreateTemplate).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/email/templates/{id:[0-9]+}", emailHandler.GetTemplate).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/email/templates/{id:[0-9]+}", emailHandler.UpdateTemplate).Methods("PUT", "OPTIONS")
	adminRouter.HandleFunc("/email/templates/{id:[0-9]+}", emailHandler.DeleteTemplate).Methods("DELETE", "OPTIONS")

	// Email usage statistics endpoint
	adminRouter.HandleFunc("/email/usage", emailHandler.GetUsage).Methods("GET", "OPTIONS")
	debug.Info("Configured admin routes: /admin/*")

	return adminRouter
}
