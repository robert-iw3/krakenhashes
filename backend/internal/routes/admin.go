package routes

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin/auth"
	adminclient "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin/client"
	adminsettings "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin/settings"
	binaryhandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/binary"
	emailhandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/email"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/middleware"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	clientsvc "github.com/ZerkerEOD/krakenhashes/backend/internal/services/client"
	retentionsvc "github.com/ZerkerEOD/krakenhashes/backend/internal/services/retention"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupAdminRoutes configures all admin-related routes
// It now accepts an AdminJobsHandler to set up job and workflow routes.
func SetupAdminRoutes(router *mux.Router, database *db.DB, emailService *email.Service, jobHandler *AdminJobsHandler) *mux.Router {
	debug.Debug("Setting up admin routes")

	// Create Repositories needed by handlers/services
	clientRepo := repository.NewClientRepository(database)
	clientSettingsRepo := repository.NewClientSettingsRepository(database)
	systemSettingsRepo := repository.NewSystemSettingsRepository(database)
	hashlistRepo := repository.NewHashListRepository(database)
	hashRepo := repository.NewHashRepository(database)

	// Create Services
	retentionService := retentionsvc.NewRetentionService(database, hashlistRepo, hashRepo, clientRepo, clientSettingsRepo)
	clientService := clientsvc.NewClientService(clientRepo, hashlistRepo, clientSettingsRepo, retentionService)

	// Get preset job repository from handler - we need to find a way to access this
	// For now, create it again here since we can't access it from the handler
	presetJobRepo := repository.NewPresetJobRepository(database.DB)

	// Create Handlers
	authSettingsHandler := auth.NewAuthSettingsHandler(database)
	emailHandler := emailhandler.NewHandler(emailService)
	retentionSettingsHandler := adminsettings.NewRetentionSettingsHandler(clientSettingsRepo)
	systemSettingsHandler := adminsettings.NewSystemSettingsHandler(systemSettingsRepo, presetJobRepo)
	jobSettingsHandler := adminsettings.NewJobSettingsHandler(systemSettingsRepo)
	clientHandler := adminclient.NewClientHandler(clientRepo, clientService)

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

	// Data Retention settings routes (New)
	adminRouter.HandleFunc("/settings/retention", retentionSettingsHandler.GetDefaultRetention).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/settings/retention", retentionSettingsHandler.UpdateDefaultRetention).Methods(http.MethodPut, http.MethodOptions)

	// System settings routes (New)
	adminRouter.HandleFunc("/settings/max-priority", systemSettingsHandler.GetMaxPriority).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/settings/max-priority", systemSettingsHandler.UpdateMaxPriority).Methods(http.MethodPut, http.MethodOptions)

	// Job execution settings routes (New)
	adminRouter.HandleFunc("/settings/job-execution", jobSettingsHandler.GetJobExecutionSettings).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/settings/job-execution", jobSettingsHandler.UpdateJobExecutionSettings).Methods(http.MethodPut, http.MethodOptions)

	// Client Management routes (New)
	adminRouter.HandleFunc("/clients", clientHandler.ListClients).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/clients", clientHandler.CreateClient).Methods(http.MethodPost, http.MethodOptions)
	adminRouter.HandleFunc("/clients/{id:[0-9a-fA-F-]+}", clientHandler.GetClient).Methods(http.MethodGet, http.MethodOptions)
	adminRouter.HandleFunc("/clients/{id:[0-9a-fA-F-]+}", clientHandler.UpdateClient).Methods(http.MethodPut, http.MethodOptions)
	adminRouter.HandleFunc("/clients/{id:[0-9a-fA-F-]+}", clientHandler.DeleteClient).Methods(http.MethodDelete, http.MethodOptions)

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

	// Binary management endpoints
	if database.DB != nil {
		binaryStore := binary.NewStore(database.DB)
		binaryManager, err := binary.NewManager(binaryStore, binary.Config{
			DataDir: config.NewConfig().DataDir,
		})
		if err != nil {
			debug.Error("Failed to initialize binary manager: %v", err)
		} else {
			binaryHandler := binaryhandler.NewHandler(binaryManager)
			adminRouter.HandleFunc("/binary", binaryHandler.HandleListVersions).Methods(http.MethodGet, http.MethodOptions)
			adminRouter.HandleFunc("/binary", binaryHandler.HandleAddVersion).Methods(http.MethodPost, http.MethodOptions)
			adminRouter.HandleFunc("/binary/{id}", binaryHandler.HandleGetVersion).Methods(http.MethodGet, http.MethodOptions)
			adminRouter.HandleFunc("/binary/{id}", binaryHandler.HandleDeleteVersion).Methods(http.MethodDelete, http.MethodOptions)
			adminRouter.HandleFunc("/binary/{id}/verify", binaryHandler.HandleVerifyVersion).Methods(http.MethodPost, http.MethodOptions)
			debug.Info("Configured admin binary management routes: /admin/binary/*")
		}
	}

	// Setup Preset Job and Job Workflow routes using the passed handler
	SetupAdminJobRoutes(adminRouter, jobHandler)
	debug.Info("Configured admin preset job and workflow routes: /admin/preset-jobs/*, /admin/job-workflows/*")

	debug.Info("Configured admin routes: /admin/*")

	return adminRouter
}
