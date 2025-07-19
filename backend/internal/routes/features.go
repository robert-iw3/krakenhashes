package routes

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/agent"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/jobs"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/pot"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/vouchers"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupDashboardRoutes configures dashboard-related routes
func SetupDashboardRoutes(jwtRouter *mux.Router) {
	jwtRouter.HandleFunc("/dashboard", dashboard.GetDashboard).Methods("GET", "OPTIONS")
	debug.Info("Configured dashboard endpoint: /dashboard")
}

// SetupJobRoutes configures job-related routes
func SetupJobRoutes(jwtRouter *mux.Router) {
	jwtRouter.HandleFunc("/jobs", jobs.GetJobs).Methods("GET", "OPTIONS")
	debug.Info("Configured jobs endpoint: /jobs")
}

// SetupAgentRoutes configures agent management routes
func SetupAgentRoutes(jwtRouter *mux.Router, agentService *services.AgentService) {
	agentHandler := agent.NewAgentHandler(agentService)
	jwtRouter.HandleFunc("/agents", agentHandler.ListAgents).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}", agentHandler.GetAgent).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}", agentHandler.UpdateAgent).Methods("PUT", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}", agentHandler.DeleteAgent).Methods("DELETE", "OPTIONS")

	// Device management routes
	jwtRouter.HandleFunc("/agents/{id}/devices", agentHandler.GetAgentDevices).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}/devices/{deviceId}", agentHandler.UpdateDeviceStatus).Methods("PUT", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}/with-devices", agentHandler.GetAgentWithDevices).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/agents/{id}/metrics", agentHandler.GetAgentMetrics).Methods("GET", "OPTIONS")

	// Force cleanup route - note: this requires admin role middleware to be added separately
	jwtRouter.HandleFunc("/agents/{id}/force-cleanup", func(w http.ResponseWriter, r *http.Request) {
		// Use the global JobIntegrationManager if available
		if JobIntegrationManager != nil && JobIntegrationManager.GetWebSocketIntegration() != nil {
			handler := admin.NewForceCleanupHandler(JobIntegrationManager.GetWebSocketIntegration())
			handler.ForceCleanup(w, r)
		} else {
			http.Error(w, "WebSocket integration not available", http.StatusServiceUnavailable)
		}
	}).Methods("POST", "OPTIONS")

	debug.Info("Configured agent management endpoints: /agents")
}

// SetupVoucherRoutes configures voucher management routes
func SetupVoucherRoutes(jwtRouter *mux.Router, voucherService *services.ClaimVoucherService) {
	voucherHandler := vouchers.NewVoucherHandler(voucherService)
	jwtRouter.HandleFunc("/vouchers/temp", voucherHandler.GenerateVoucher).Methods("POST", "OPTIONS")
	jwtRouter.HandleFunc("/vouchers", voucherHandler.ListVouchers).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/vouchers/{code}/disable", voucherHandler.DeactivateVoucher).Methods("DELETE", "OPTIONS")
	debug.Info("Configured voucher management endpoints: /vouchers")
}

// SetupPotRoutes configures pot (cracked hashes) routes
func SetupPotRoutes(jwtRouter *mux.Router, hashRepo *repository.HashRepository, hashlistRepo *repository.HashListRepository, clientRepo *repository.ClientRepository) {
	potHandler := pot.NewHandler(hashRepo, hashlistRepo, clientRepo)
	
	// List routes
	jwtRouter.HandleFunc("/pot", potHandler.HandleListCrackedHashes).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/hashlist/{id}", potHandler.HandleListCrackedHashesByHashlist).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/client/{id}", potHandler.HandleListCrackedHashesByClient).Methods("GET", "OPTIONS")
	
	// Download routes for all cracked hashes
	jwtRouter.HandleFunc("/pot/download/hash-pass", potHandler.HandleDownloadHashPass).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/download/user-pass", potHandler.HandleDownloadUserPass).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/download/user", potHandler.HandleDownloadUser).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/download/pass", potHandler.HandleDownloadPass).Methods("GET", "OPTIONS")
	
	// Download routes for hashlist-specific cracked hashes
	jwtRouter.HandleFunc("/pot/hashlist/{id}/download/hash-pass", potHandler.HandleDownloadHashPassByHashlist).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/hashlist/{id}/download/user-pass", potHandler.HandleDownloadUserPassByHashlist).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/hashlist/{id}/download/user", potHandler.HandleDownloadUserByHashlist).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/hashlist/{id}/download/pass", potHandler.HandleDownloadPassByHashlist).Methods("GET", "OPTIONS")
	
	// Download routes for client-specific cracked hashes
	jwtRouter.HandleFunc("/pot/client/{id}/download/hash-pass", potHandler.HandleDownloadHashPassByClient).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/client/{id}/download/user-pass", potHandler.HandleDownloadUserPassByClient).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/client/{id}/download/user", potHandler.HandleDownloadUserByClient).Methods("GET", "OPTIONS")
	jwtRouter.HandleFunc("/pot/client/{id}/download/pass", potHandler.HandleDownloadPassByClient).Methods("GET", "OPTIONS")
	
	debug.Info("Configured pot endpoints: list and download routes for all/hashlist/client contexts")
}
