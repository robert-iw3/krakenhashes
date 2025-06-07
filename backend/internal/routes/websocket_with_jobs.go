package routes

import (
	cryptotls "crypto/tls"
	"database/sql"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/auth/api"
	wshandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/integration"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SetupWebSocketWithJobRoutes configures WebSocket routes with job execution integration
func SetupWebSocketWithJobRoutes(
	r *mux.Router,
	agentService *services.AgentService,
	tlsProvider tls.Provider,
	sqlDB *sql.DB,
	appConfig *config.Config,
) {
	debug.Debug("Setting up WebSocket routes with job integration")
	
	// Create database wrapper
	database := &db.DB{DB: sqlDB}
	
	// Create repositories
	benchmarkRepo := repository.NewBenchmarkRepository(database)
	presetJobRepo := repository.NewPresetJobRepository(sqlDB)
	hashlistRepo := repository.NewHashListRepository(database)
	jobTaskRepo := repository.NewJobTaskRepository(database)
	agentRepo := repository.NewAgentRepository(database)
	jobExecutionRepo := repository.NewJobExecutionRepository(database)
	systemSettingsRepo := repository.NewSystemSettingsRepository(database)
	agentHashlistRepo := repository.NewAgentHashlistRepository(database)
	
	// Create services
	jobExecutionService := services.NewJobExecutionService(
		jobExecutionRepo,
		jobTaskRepo,
		benchmarkRepo,
		agentHashlistRepo,
		agentRepo,
		presetJobRepo,
		hashlistRepo,
		systemSettingsRepo,
		"/usr/bin/hashcat", // hashcat binary path
		appConfig.DataDir,
	)
	
	jobChunkingService := services.NewJobChunkingService(
		benchmarkRepo,
		jobTaskRepo,
		systemSettingsRepo,
	)
	
	hashlistSyncService := services.NewHashlistSyncService(
		agentHashlistRepo,
		hashlistRepo,
		systemSettingsRepo,
		appConfig.DataDir,
	)
	
	jobSchedulingService := services.NewJobSchedulingService(
		jobExecutionService,
		jobChunkingService,
		hashlistSyncService,
		agentRepo,
		systemSettingsRepo,
	)
	
	// Create WebSocket service
	wsService := wsservice.NewService(agentService)
	
	// Get TLS configuration for WebSocket handler
	tlsConfig, err := tlsProvider.GetTLSConfig()
	if err != nil {
		debug.Error("Failed to get TLS configuration: %v", err)
		return
	}
	
	// Create a new TLS config for agent connections
	agentTLSConfig := &cryptotls.Config{
		Certificates:             tlsConfig.Certificates,
		RootCAs:                  tlsConfig.RootCAs,
		MinVersion:               tlsConfig.MinVersion,
		MaxVersion:               tlsConfig.MaxVersion,
		CipherSuites:             tlsConfig.CipherSuites,
		PreferServerCipherSuites: true,
	}
	
	// Create WebSocket handler
	wsHandler := wshandler.NewHandler(wsService, agentService, agentTLSConfig)
	
	// Create job integration manager
	jobIntegration := integration.NewJobIntegrationManager(
		wsHandler,
		jobSchedulingService,
		jobExecutionService,
		hashlistSyncService,
		benchmarkRepo,
		presetJobRepo,
		hashlistRepo,
		jobTaskRepo,
		agentRepo,
	)
	
	// Set the job handler in the WebSocket service
	wsService.SetJobHandler(jobIntegration)
	
	// Setup WebSocket routes
	wsRouter := r.PathPrefix("/ws").Subrouter()
	wsRouter.Use(api.APIKeyMiddleware(agentService))
	wsRouter.Use(loggingMiddleware)
	
	wsRouter.HandleFunc("/agent", wsHandler.ServeWS)
	debug.Info("Configured WebSocket endpoint: /ws/agent with job integration and TLS: %v", tlsConfig != nil)
	
	// Start the job scheduler
	// TODO: This should be started with a proper context from main()
	// For now, we'll comment it out until we can properly manage the lifecycle
	// jobIntegration.StartScheduler(context.Background())
	
	if tlsConfig != nil {
		debug.Debug("WebSocket TLS Configuration:")
		debug.Debug("- Min Version: %v", agentTLSConfig.MinVersion)
		debug.Debug("- Certificates: %d", len(agentTLSConfig.Certificates))
	}
}