package routes

import (
	"database/sql"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/auth/api"
	binaryhandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupBinaryRoutes configures all binary management related routes
func SetupBinaryRoutes(r *mux.Router, sqlDB *sql.DB, cfg *config.Config, agentService *services.AgentService) {
	debug.Info("Setting up binary management routes")

	// Initialize binary store and manager
	store := binary.NewStore(sqlDB)
	manager, err := binary.NewManager(store, binary.Config{
		DataDir: cfg.DataDir,
	})
	if err != nil {
		debug.Error("Failed to initialize binary manager: %v", err)
		return
	}

	// Create handler
	handler := binaryhandler.NewHandler(manager)

	// Agent routes (protected by agent authentication)
	// These routes are for agents to download binaries and should use agent authentication
	agentRouter := r.PathPrefix("/api/binary").Subrouter()
	agentRouter.Use(api.APIKeyMiddleware(agentService))

	// Register agent routes
	agentRouter.HandleFunc("/latest", handler.HandleGetLatestVersion).Methods(http.MethodGet, http.MethodOptions)
	agentRouter.HandleFunc("/download/{id}", handler.HandleDownloadBinary).Methods(http.MethodGet, http.MethodOptions)
	debug.Info("Registered agent binary management routes")
}
