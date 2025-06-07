package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// JobIntegrationManager manages the integration between WebSocket and job execution services
type JobIntegrationManager struct {
	wsIntegration        *JobWebSocketIntegration
	jobSchedulingService *services.JobSchedulingService
	wsHandler            interface {
		SendMessage(agentID int, msg *wsservice.Message) error
		GetConnectedAgents() []int
	}
}

// NewJobIntegrationManager creates a new job integration manager
func NewJobIntegrationManager(
	wsHandler interface {
		SendMessage(agentID int, msg *wsservice.Message) error
		GetConnectedAgents() []int
	},
	jobSchedulingService *services.JobSchedulingService,
	jobExecutionService *services.JobExecutionService,
	hashlistSyncService *services.HashlistSyncService,
	benchmarkRepo *repository.BenchmarkRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	jobTaskRepo *repository.JobTaskRepository,
	agentRepo *repository.AgentRepository,
) *JobIntegrationManager {
	// Create the WebSocket integration
	wsIntegration := NewJobWebSocketIntegration(
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

	// Set the WebSocket integration in the scheduling service
	jobSchedulingService.SetWebSocketIntegration(wsIntegration)

	return &JobIntegrationManager{
		wsIntegration:        wsIntegration,
		jobSchedulingService: jobSchedulingService,
		wsHandler:            wsHandler,
	}
}

// ProcessJobProgress handles job progress messages from agents (implements interfaces.JobHandler)
func (m *JobIntegrationManager) ProcessJobProgress(ctx context.Context, agentID int, payload json.RawMessage) error {
	var progress models.JobProgress
	if err := json.Unmarshal(payload, &progress); err != nil {
		return fmt.Errorf("failed to unmarshal job progress: %w", err)
	}

	return m.wsIntegration.HandleJobProgress(ctx, agentID, &progress)
}

// ProcessBenchmarkResult handles benchmark result messages from agents (implements interfaces.JobHandler)
func (m *JobIntegrationManager) ProcessBenchmarkResult(ctx context.Context, agentID int, payload json.RawMessage) error {
	var result wsservice.BenchmarkResultPayload
	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("failed to unmarshal benchmark result: %w", err)
	}

	return m.wsIntegration.HandleBenchmarkResult(ctx, agentID, &result)
}

// StartScheduler starts the job scheduling service
func (m *JobIntegrationManager) StartScheduler(ctx context.Context) {
	debug.Log("Starting job scheduler", nil)
	// Start scheduler with 30 second interval
	go m.jobSchedulingService.StartScheduler(ctx, 30*time.Second)
}

// StopJob stops a running job (TODO: implement in JobSchedulingService)
func (m *JobIntegrationManager) StopJob(ctx context.Context, jobExecutionID uuid.UUID, reason string) error {
	debug.Log("Stop job requested", map[string]interface{}{
		"job_execution_id": jobExecutionID,
		"reason":          reason,
	})
	// TODO: Implement stop job functionality in JobSchedulingService
	return fmt.Errorf("stop job functionality not yet implemented")
}

// GetConnectedAgentCount returns the number of connected agents
func (m *JobIntegrationManager) GetConnectedAgentCount() int {
	return len(m.wsHandler.GetConnectedAgents())
}