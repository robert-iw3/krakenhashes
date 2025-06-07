package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// JobWebSocketIntegration handles the integration between job scheduling and WebSocket communication
type JobWebSocketIntegration struct {
	wsHandler           interface{ SendMessage(agentID int, msg *wsservice.Message) error }
	jobSchedulingService *services.JobSchedulingService
	jobExecutionService  *services.JobExecutionService
	hashlistSyncService  *services.HashlistSyncService
	benchmarkRepo        *repository.BenchmarkRepository
	presetJobRepo        repository.PresetJobRepository
	hashlistRepo         *repository.HashListRepository
	jobTaskRepo          *repository.JobTaskRepository
	agentRepo            *repository.AgentRepository
	
	// Progress tracking
	progressMutex    sync.RWMutex
	taskProgressMap  map[string]*models.JobProgress // TaskID -> Progress
}

// NewJobWebSocketIntegration creates a new job WebSocket integration service
func NewJobWebSocketIntegration(
	wsHandler interface{ SendMessage(agentID int, msg *wsservice.Message) error },
	jobSchedulingService *services.JobSchedulingService,
	jobExecutionService *services.JobExecutionService,
	hashlistSyncService *services.HashlistSyncService,
	benchmarkRepo *repository.BenchmarkRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	jobTaskRepo *repository.JobTaskRepository,
	agentRepo *repository.AgentRepository,
) *JobWebSocketIntegration {
	return &JobWebSocketIntegration{
		wsHandler:            wsHandler,
		jobSchedulingService: jobSchedulingService,
		jobExecutionService:  jobExecutionService,
		hashlistSyncService:  hashlistSyncService,
		benchmarkRepo:        benchmarkRepo,
		presetJobRepo:        presetJobRepo,
		hashlistRepo:         hashlistRepo,
		jobTaskRepo:          jobTaskRepo,
		agentRepo:            agentRepo,
		taskProgressMap:      make(map[string]*models.JobProgress),
	}
}

// SendJobAssignment sends a job task assignment to an agent via WebSocket
func (s *JobWebSocketIntegration) SendJobAssignment(ctx context.Context, task *models.JobTask, jobExecution *models.JobExecution) error {
	debug.Log("Sending job assignment to agent", map[string]interface{}{
		"task_id":   task.ID,
		"agent_id":  task.AgentID,
		"job_id":    jobExecution.ID,
	})

	// Get preset job details
	presetJob, err := s.presetJobRepo.GetByID(ctx, jobExecution.PresetJobID)
	if err != nil {
		return fmt.Errorf("failed to get preset job: %w", err)
	}

	// Get hashlist details
	hashlist, err := s.hashlistRepo.GetByID(ctx, jobExecution.HashlistID)
	if err != nil {
		return fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Get agent details to find agent int ID
	agent, err := s.agentRepo.GetByID(ctx, task.AgentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Build wordlist and rule paths
	var wordlistPaths []string
	for _, wordlistIDStr := range presetJob.WordlistIDs {
		wordlistPath := fmt.Sprintf("wordlists/custom/%s.txt", wordlistIDStr)
		wordlistPaths = append(wordlistPaths, wordlistPath)
	}

	var rulePaths []string
	for _, ruleIDStr := range presetJob.RuleIDs {
		rulePath := fmt.Sprintf("rules/custom/%s.rule", ruleIDStr)
		rulePaths = append(rulePaths, rulePath)
	}

	// Get binary path from binary version
	binaryPath := fmt.Sprintf("binaries/hashcat_%d", presetJob.BinaryVersionID)

	// Get report interval from settings or use default
	reportInterval := 5 // Default 5 seconds
	if val, err := s.jobExecutionService.GetSystemSetting(ctx, "progress_reporting_interval"); err == nil {
		reportInterval = val
	}

	// Create task assignment payload
	assignment := wsservice.TaskAssignmentPayload{
		TaskID:          task.ID.String(),
		JobExecutionID:  jobExecution.ID.String(),
		HashlistID:      jobExecution.HashlistID,
		HashlistPath:    fmt.Sprintf("hashlists/%d.hash", jobExecution.HashlistID),
		AttackMode:      int(jobExecution.AttackMode),
		HashType:        hashlist.HashTypeID,
		KeyspaceStart:   task.KeyspaceStart,
		KeyspaceEnd:     task.KeyspaceEnd,
		WordlistPaths:   wordlistPaths,
		RulePaths:       rulePaths,
		Mask:            presetJob.Mask,
		BinaryPath:      binaryPath,
		ChunkDuration:   task.ChunkDuration,
		ReportInterval:  reportInterval,
		OutputFormat:    "3", // hash:plain format
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("failed to marshal task assignment: %w", err)
	}

	// Create WebSocket message
	msg := &wsservice.Message{
		Type:    wsservice.TypeTaskAssignment,
		Payload: payloadBytes,
	}

	// Send via WebSocket
	err = s.wsHandler.SendMessage(agent.ID, msg)
	if err != nil {
		return fmt.Errorf("failed to send task assignment via WebSocket: %w", err)
	}

	// Update task status to assigned
	err = s.jobTaskRepo.UpdateStatus(ctx, task.ID, models.JobTaskStatusAssigned)
	if err != nil {
		debug.Log("Failed to update task status to assigned", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	}

	debug.Log("Job assignment sent successfully", map[string]interface{}{
		"task_id":  task.ID,
		"agent_id": agent.ID,
	})

	return nil
}

// SendJobStop sends a stop command for a job to an agent
func (s *JobWebSocketIntegration) SendJobStop(ctx context.Context, taskID uuid.UUID, reason string) error {
	// Get task details
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get agent details
	agent, err := s.agentRepo.GetByID(ctx, task.AgentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	debug.Log("Sending job stop command to agent", map[string]interface{}{
		"task_id":  taskID,
		"agent_id": agent.ID,
		"reason":   reason,
	})

	// Create stop payload
	stopPayload := wsservice.JobStopPayload{
		TaskID: taskID.String(),
		Reason: reason,
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(stopPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal job stop: %w", err)
	}

	// Create WebSocket message
	msg := &wsservice.Message{
		Type:    wsservice.TypeJobStop,
		Payload: payloadBytes,
	}

	// Send via WebSocket
	err = s.wsHandler.SendMessage(agent.ID, msg)
	if err != nil {
		return fmt.Errorf("failed to send job stop via WebSocket: %w", err)
	}

	debug.Log("Job stop command sent successfully", map[string]interface{}{
		"task_id":  taskID,
		"agent_id": agent.ID,
	})

	return nil
}

// SendBenchmarkRequest sends a benchmark request to an agent
func (s *JobWebSocketIntegration) SendBenchmarkRequest(ctx context.Context, agentID int, hashType int, attackMode models.AttackMode, binaryVersionID int) error {
	// Get agent details
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	requestID := fmt.Sprintf("benchmark-%s-%d-%d-%d", agentID, hashType, attackMode, time.Now().Unix())
	binaryPath := fmt.Sprintf("binaries/hashcat_%d", binaryVersionID)

	debug.Log("Sending benchmark request to agent", map[string]interface{}{
		"agent_id":    agentID,
		"hash_type":   hashType,
		"attack_mode": attackMode,
		"request_id":  requestID,
	})

	// Create benchmark request payload
	benchmarkReq := wsservice.BenchmarkRequestPayload{
		RequestID:  requestID,
		HashType:   hashType,
		AttackMode: int(attackMode),
		BinaryPath: binaryPath,
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(benchmarkReq)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark request: %w", err)
	}

	// Create WebSocket message
	msg := &wsservice.Message{
		Type:    wsservice.TypeBenchmarkRequest,
		Payload: payloadBytes,
	}

	// Send via WebSocket
	err = s.wsHandler.SendMessage(agent.ID, msg)
	if err != nil {
		return fmt.Errorf("failed to send benchmark request via WebSocket: %w", err)
	}

	debug.Log("Benchmark request sent successfully", map[string]interface{}{
		"agent_id":   agentID,
		"request_id": requestID,
	})

	return nil
}

// HandleJobProgress processes job progress updates from agents
func (s *JobWebSocketIntegration) HandleJobProgress(ctx context.Context, agentID int, progress *models.JobProgress) error {
	debug.Log("Processing job progress from agent", map[string]interface{}{
		"agent_id":           agentID,
		"task_id":            progress.TaskID,
		"keyspace_processed": progress.KeyspaceProcessed,
		"hash_rate":          progress.HashRate,
	})

	// Store progress in memory
	s.progressMutex.Lock()
	s.taskProgressMap[progress.TaskID.String()] = progress
	s.progressMutex.Unlock()

	// Forward to job scheduling service
	err := s.jobSchedulingService.ProcessTaskProgress(ctx, progress.TaskID, progress)
	if err != nil {
		return fmt.Errorf("failed to process task progress: %w", err)
	}

	// Process cracked hashes if any
	if progress.CrackedCount > 0 && len(progress.CrackedHashes) > 0 {
		err = s.processCrackedHashes(ctx, progress.TaskID, progress.CrackedHashes)
		if err != nil {
			debug.Log("Failed to process cracked hashes", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
		}
	}

	// Check if task is complete
	task, err := s.jobTaskRepo.GetByID(ctx, progress.TaskID)
	if err == nil {
		if progress.KeyspaceProcessed >= (task.KeyspaceEnd - task.KeyspaceStart) {
			// Task is complete
			err = s.jobTaskRepo.CompleteTask(ctx, progress.TaskID)
			if err != nil {
				debug.Log("Failed to mark task as complete", map[string]interface{}{
					"task_id": progress.TaskID,
					"error":   err.Error(),
				})
			}

			// Check if job is complete
			err = s.jobSchedulingService.ProcessJobCompletion(ctx, task.JobExecutionID)
			if err != nil {
				debug.Log("Failed to process job completion", map[string]interface{}{
					"job_execution_id": task.JobExecutionID,
					"error":            err.Error(),
				})
			}
		}
	}

	return nil
}

// HandleBenchmarkResult processes benchmark results from agents
func (s *JobWebSocketIntegration) HandleBenchmarkResult(ctx context.Context, agentID int, result *wsservice.BenchmarkResultPayload) error {
	debug.Log("Processing benchmark result from agent", map[string]interface{}{
		"agent_id":    agentID,
		"hash_type":   result.HashType,
		"attack_mode": result.AttackMode,
		"speed":       result.Speed,
		"success":     result.Success,
	})

	if !result.Success {
		debug.Log("Benchmark failed", map[string]interface{}{
			"agent_id": agentID,
			"error":    result.Error,
		})
		return fmt.Errorf("benchmark failed: %s", result.Error)
	}

	// Get agent
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Store benchmark result
	benchmark := &models.AgentBenchmark{
		AgentID:    agent.ID,
		AttackMode: models.AttackMode(result.AttackMode),
		HashType:   result.HashType,
		Speed:      result.Speed,
	}

	err = s.benchmarkRepo.CreateOrUpdateAgentBenchmark(ctx, benchmark)
	if err != nil {
		return fmt.Errorf("failed to store benchmark result: %w", err)
	}

	debug.Log("Benchmark result stored successfully", map[string]interface{}{
		"agent_id":    agentID,
		"hash_type":   result.HashType,
		"attack_mode": result.AttackMode,
		"speed":       result.Speed,
	})

	return nil
}

// processCrackedHashes processes cracked hashes from a job progress update
func (s *JobWebSocketIntegration) processCrackedHashes(ctx context.Context, taskID uuid.UUID, crackedHashes []string) error {
	// Get task details
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get job execution details
	jobExecution, err := s.jobExecutionService.GetJobExecutionByID(ctx, task.JobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get job execution: %w", err)
	}

	// Process each cracked hash
	for _, crackedEntry := range crackedHashes {
		// Parse hash:plain format
		// This is a simplified version - actual implementation would need proper parsing
		debug.Log("Processing cracked hash", map[string]interface{}{
			"hashlist_id": jobExecution.HashlistID,
			"entry":       crackedEntry,
		})

		// TODO: Forward to crack update handler
		// This would integrate with the existing crack update WebSocket endpoint
	}

	// Update hashlist file to remove cracked hashes
	err = s.hashlistSyncService.UpdateHashlistAfterCracks(ctx, jobExecution.HashlistID, crackedHashes)
	if err != nil {
		debug.Log("Failed to update hashlist after cracks", map[string]interface{}{
			"hashlist_id": jobExecution.HashlistID,
			"error":       err.Error(),
		})
	}

	return nil
}

// GetTaskProgress returns the current progress for a task
func (s *JobWebSocketIntegration) GetTaskProgress(taskID string) *models.JobProgress {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()
	
	return s.taskProgressMap[taskID]
}

// StartScheduledJobAssignment starts the process of assigning scheduled jobs to agents
func (s *JobWebSocketIntegration) StartScheduledJobAssignment(ctx context.Context) {
	// This would be called when the scheduling service assigns a task to an agent
	// The scheduling service would call SendJobAssignment for each assigned task
	debug.Log("Job assignment integration service started", nil)
}