package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)


// JobWebSocketIntegration interface for WebSocket integration
type JobWebSocketIntegration interface {
	SendJobAssignment(ctx context.Context, task *models.JobTask, jobExecution *models.JobExecution) error
}

// JobSchedulingService handles the assignment of jobs to agents
type JobSchedulingService struct {
	jobExecutionService *JobExecutionService
	jobChunkingService  *JobChunkingService
	hashlistSyncService *HashlistSyncService
	agentRepo           *repository.AgentRepository
	systemSettingsRepo  *repository.SystemSettingsRepository
	wsIntegration       JobWebSocketIntegration
	
	// Scheduling state
	schedulingMutex sync.Mutex
	isScheduling    bool
}

// NewJobSchedulingService creates a new job scheduling service
func NewJobSchedulingService(
	jobExecutionService *JobExecutionService,
	jobChunkingService *JobChunkingService,
	hashlistSyncService *HashlistSyncService,
	agentRepo *repository.AgentRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
) *JobSchedulingService {
	return &JobSchedulingService{
		jobExecutionService: jobExecutionService,
		jobChunkingService:  jobChunkingService,
		hashlistSyncService: hashlistSyncService,
		agentRepo:           agentRepo,
		systemSettingsRepo:  systemSettingsRepo,
	}
}

// ScheduleJobsResult contains the result of a scheduling operation
type ScheduleJobsResult struct {
	AssignedTasks   []models.JobTask
	InterruptedJobs []uuid.UUID
	Errors          []error
}

// ScheduleJobs performs the main job scheduling logic
func (s *JobSchedulingService) ScheduleJobs(ctx context.Context) (*ScheduleJobsResult, error) {
	s.schedulingMutex.Lock()
	defer s.schedulingMutex.Unlock()

	if s.isScheduling {
		return nil, fmt.Errorf("scheduling already in progress")
	}

	s.isScheduling = true
	defer func() { s.isScheduling = false }()

	debug.Log("Starting job scheduling cycle", nil)

	result := &ScheduleJobsResult{
		AssignedTasks:   []models.JobTask{},
		InterruptedJobs: []uuid.UUID{},
		Errors:          []error{},
	}

	// Get available agents
	availableAgents, err := s.jobExecutionService.GetAvailableAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	if len(availableAgents) == 0 {
		debug.Log("No available agents for job scheduling", nil)
		return result, nil
	}

	debug.Log("Found available agents", map[string]interface{}{
		"agent_count": len(availableAgents),
	})

	// Process each available agent
	for _, agent := range availableAgents {
		taskAssigned, interruptedJobs, err := s.assignWorkToAgent(ctx, &agent)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to assign work to agent %s: %w", agent.ID, err))
			continue
		}

		if taskAssigned != nil {
			result.AssignedTasks = append(result.AssignedTasks, *taskAssigned)
		}

		result.InterruptedJobs = append(result.InterruptedJobs, interruptedJobs...)
	}

	debug.Log("Job scheduling cycle completed", map[string]interface{}{
		"assigned_tasks":   len(result.AssignedTasks),
		"interrupted_jobs": len(result.InterruptedJobs),
		"errors":           len(result.Errors),
	})

	return result, nil
}

// assignWorkToAgent assigns work to a specific agent
func (s *JobSchedulingService) assignWorkToAgent(ctx context.Context, agent *models.Agent) (*models.JobTask, []uuid.UUID, error) {
	debug.Log("Assigning work to agent", map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
	})

	// Get the next pending job (priority + FIFO)
	nextJob, err := s.jobExecutionService.GetNextPendingJob(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get next pending job: %w", err)
	}

	if nextJob == nil {
		debug.Log("No pending jobs for agent", map[string]interface{}{
			"agent_id": agent.ID,
		})
		return nil, nil, nil // No work available
	}

	// Check if we need to interrupt any running jobs for higher priority
	var interruptedJobs []uuid.UUID
	interruptibleJobs, err := s.jobExecutionService.CanInterruptJob(ctx, nextJob.Priority)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check interruptible jobs: %w", err)
	}

	// If we have interruptible jobs and the new job has higher priority, interrupt them
	if len(interruptibleJobs) > 0 {
		for _, interruptibleJob := range interruptibleJobs {
			err = s.jobExecutionService.InterruptJob(ctx, interruptibleJob.ID, nextJob.ID)
			if err != nil {
				debug.Log("Failed to interrupt job", map[string]interface{}{
					"job_id": interruptibleJob.ID,
					"error":  err.Error(),
				})
				continue
			}
			interruptedJobs = append(interruptedJobs, interruptibleJob.ID)
		}
	}

	// Ensure the hashlist is available on the agent
	err = s.hashlistSyncService.EnsureHashlistOnAgent(ctx, agent.ID, nextJob.HashlistID)
	if err != nil {
		return nil, interruptedJobs, fmt.Errorf("failed to sync hashlist to agent: %w", err)
	}

	// Calculate the next chunk for this agent
	chunkReq := ChunkCalculationRequest{
		JobExecution:  nextJob,
		Agent:         agent,
		AttackMode:    nextJob.AttackMode,
		HashType:      0, // This should come from the hashlist
		ChunkDuration: 1200, // This should come from settings or preset job
	}

	// Get chunk duration from settings or preset job
	if chunkDuration, err := s.getChunkDuration(ctx, nextJob); err == nil {
		chunkReq.ChunkDuration = chunkDuration
	}

	chunkResult, err := s.jobChunkingService.CalculateNextChunk(ctx, chunkReq)
	if err != nil {
		return nil, interruptedJobs, fmt.Errorf("failed to calculate chunk: %w", err)
	}

	// Create the job task
	jobTask, err := s.jobExecutionService.CreateJobTask(
		ctx,
		nextJob,
		agent,
		chunkResult.KeyspaceStart,
		chunkResult.KeyspaceEnd,
		chunkResult.BenchmarkSpeed,
	)
	if err != nil {
		return nil, interruptedJobs, fmt.Errorf("failed to create job task: %w", err)
	}

	// Start the job execution if this is the first task
	if chunkResult.KeyspaceStart == 0 {
		err = s.jobExecutionService.StartJobExecution(ctx, nextJob.ID)
		if err != nil {
			debug.Log("Failed to start job execution", map[string]interface{}{
				"job_execution_id": nextJob.ID,
				"error":            err.Error(),
			})
		}
	}

	// Send the task assignment via WebSocket if integration is available
	if s.wsIntegration != nil {
		err = s.wsIntegration.SendJobAssignment(ctx, jobTask, nextJob)
		if err != nil {
			// Log error but don't fail the assignment - the agent can still poll for work
			debug.Log("Failed to send job assignment via WebSocket", map[string]interface{}{
				"task_id": jobTask.ID,
				"error":   err.Error(),
			})
		}
	}

	debug.Log("Work assigned to agent", map[string]interface{}{
		"agent_id":        agent.ID,
		"job_task_id":     jobTask.ID,
		"job_execution_id": nextJob.ID,
		"keyspace_start":  chunkResult.KeyspaceStart,
		"keyspace_end":    chunkResult.KeyspaceEnd,
	})

	return jobTask, interruptedJobs, nil
}

// getChunkDuration gets the chunk duration for a job from preset job or settings
func (s *JobSchedulingService) getChunkDuration(ctx context.Context, jobExecution *models.JobExecution) (int, error) {
	// First try to get from preset job
	presetJob, err := s.jobExecutionService.presetJobRepo.GetByID(ctx, jobExecution.PresetJobID)
	if err == nil && presetJob.ChunkSizeSeconds > 0 {
		return presetJob.ChunkSizeSeconds, nil
	}

	// Fall back to system setting
	setting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		return 1200, nil // Default 20 minutes
	}

	chunkDuration := 1200 // Default 20 minutes
	if setting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*setting.Value); parseErr == nil {
			chunkDuration = parsed
		}
	}

	return chunkDuration, nil
}

// StartScheduler starts the job scheduler with periodic scheduling
func (s *JobSchedulingService) StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	debug.Log("Job scheduler started", map[string]interface{}{
		"interval": interval,
	})

	for {
		select {
		case <-ctx.Done():
			debug.Log("Job scheduler stopped", nil)
			return
		case <-ticker.C:
			result, err := s.ScheduleJobs(ctx)
			if err != nil {
				debug.Log("Scheduling cycle failed", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			// Log scheduling results
			if len(result.AssignedTasks) > 0 || len(result.InterruptedJobs) > 0 || len(result.Errors) > 0 {
				debug.Log("Scheduling cycle completed", map[string]interface{}{
					"assigned_tasks":   len(result.AssignedTasks),
					"interrupted_jobs": len(result.InterruptedJobs),
					"errors":           len(result.Errors),
				})
			}
		}
	}
}

// ProcessJobCompletion handles job completion and cleanup
func (s *JobSchedulingService) ProcessJobCompletion(ctx context.Context, jobExecutionID uuid.UUID) error {
	debug.Log("Processing job completion", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	// Check if all tasks for this job are completed
	incompleteTasks, err := s.jobExecutionService.jobTaskRepo.GetIncompleteTasksCount(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get incomplete tasks count: %w", err)
	}

	if incompleteTasks == 0 {
		// All tasks are complete, mark job as completed
		err = s.jobExecutionService.CompleteJobExecution(ctx, jobExecutionID)
		if err != nil {
			return fmt.Errorf("failed to complete job execution: %w", err)
		}

		debug.Log("Job execution completed", map[string]interface{}{
			"job_execution_id": jobExecutionID,
		})
	}

	return nil
}

// ProcessTaskProgress handles task progress updates and job aggregation
func (s *JobSchedulingService) ProcessTaskProgress(ctx context.Context, taskID uuid.UUID, progress *models.JobProgress) error {
	// Update task progress
	err := s.jobExecutionService.jobTaskRepo.UpdateProgress(ctx, taskID, progress.KeyspaceProcessed, &progress.HashRate)
	if err != nil {
		return fmt.Errorf("failed to update task progress: %w", err)
	}

	// Get the task to find the job execution
	task, err := s.jobExecutionService.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Aggregate progress for the entire job
	tasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByJobExecution(ctx, task.JobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get all tasks for job: %w", err)
	}

	var totalProcessed int64
	for _, t := range tasks {
		totalProcessed += t.KeyspaceProcessed
	}

	// Update job execution progress
	err = s.jobExecutionService.UpdateJobProgress(ctx, task.JobExecutionID, totalProcessed)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	// Store performance metrics
	if progress.HashRate > 0 {
		metric := &models.JobPerformanceMetric{
			JobExecutionID:   task.JobExecutionID,
			MetricType:       models.JobMetricTypeHashRate,
			Value:            float64(progress.HashRate),
			Timestamp:        time.Now(),
			AggregationLevel: models.AggregationLevelRealtime,
		}

		err = s.jobExecutionService.benchmarkRepo.CreateJobPerformanceMetric(ctx, metric)
		if err != nil {
			debug.Log("Failed to store job performance metric", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// GetJobExecutionStatus returns the current status of a job execution
func (s *JobSchedulingService) GetJobExecutionStatus(ctx context.Context, jobExecutionID uuid.UUID) (*models.JobExecution, []models.JobTask, error) {
	jobExecution, err := s.jobExecutionService.jobExecRepo.GetByID(ctx, jobExecutionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get job execution: %w", err)
	}

	tasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get job tasks: %w", err)
	}

	return jobExecution, tasks, nil
}

// SetWebSocketIntegration sets the WebSocket integration for sending job assignments
func (s *JobSchedulingService) SetWebSocketIntegration(integration JobWebSocketIntegration) {
	s.wsIntegration = integration
}

