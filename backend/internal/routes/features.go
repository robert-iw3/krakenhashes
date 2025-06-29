package routes

import (
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/agent"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/jobs"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/vouchers"
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
