package routes

import (
	cryptotls "crypto/tls"
	"database/sql"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/auth/api"
	wshandler "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/integration"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/rule"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// JobIntegrationManager is a global reference to the job integration manager
// This is a temporary solution until we refactor main to properly manage lifecycle
var JobIntegrationManager *integration.JobIntegrationManager

// SetupWebSocketWithJobRoutes configures WebSocket routes with job execution integration
func SetupWebSocketWithJobRoutes(
	r *mux.Router,
	agentService *services.AgentService,
	tlsProvider tls.Provider,
	sqlDB *sql.DB,
	appConfig *config.Config,
	wordlistManager wordlist.Manager,
	ruleManager rule.Manager,
	binaryManager binary.Manager,
) {
	debug.Debug("Setting up WebSocket routes with job integration")
	
	// Create database wrapper
	database := &db.DB{DB: sqlDB}
	
	// Create repositories
	benchmarkRepo := repository.NewBenchmarkRepository(database)
	presetJobRepo := repository.NewPresetJobRepository(sqlDB)
	hashlistRepo := repository.NewHashListRepository(database)
	hashRepo := repository.NewHashRepository(database)
	jobTaskRepo := repository.NewJobTaskRepository(database)
	agentRepo := repository.NewAgentRepository(database)
	jobExecutionRepo := repository.NewJobExecutionRepository(database)
	systemSettingsRepo := repository.NewSystemSettingsRepository(database)
	agentHashlistRepo := repository.NewAgentHashlistRepository(database)
	fileRepo := repository.NewFileRepository(database, appConfig.DataDir)
	
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
		fileRepo,
		binaryManager,
		"/usr/bin/hashcat", // hashcat binary path (deprecated, using binary manager now)
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
		hashRepo,
		jobTaskRepo,
		agentRepo,
		sqlDB,
		wordlistManager,
		ruleManager,
		binaryManager,
	)
	
	// Set the job handler in the WebSocket service
	wsService.SetJobHandler(jobIntegration)
	
	// Setup WebSocket routes
	wsRouter := r.PathPrefix("/ws").Subrouter()
	wsRouter.Use(api.APIKeyMiddleware(agentService))
	wsRouter.Use(loggingMiddleware)
	
	wsRouter.HandleFunc("/agent", wsHandler.ServeWS)
	debug.Info("Configured WebSocket endpoint: /ws/agent with job integration and TLS: %v", tlsConfig != nil)
	
	// Return the job integration manager so it can be started from main
	// Store it globally for now until we refactor the main function
	JobIntegrationManager = jobIntegration
	
	if tlsConfig != nil {
		debug.Debug("WebSocket TLS Configuration:")
		debug.Debug("- Min Version: %v", agentTLSConfig.MinVersion)
		debug.Debug("- Certificates: %d", len(agentTLSConfig.Certificates))
	}
}