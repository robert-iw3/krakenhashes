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
	RequestAgentBenchmark(ctx context.Context, agentID int, jobExecution *models.JobExecution) error
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
// The function now checks if the agent has a valid benchmark for the job's attack mode and hash type.
// If no benchmark exists or it's outdated, it requests a benchmark from the agent and defers the job assignment.
// This ensures accurate chunk calculations based on real-world performance.
func (s *JobSchedulingService) assignWorkToAgent(ctx context.Context, agent *models.Agent) (*models.JobTask, []uuid.UUID, error) {
	debug.Log("Assigning work to agent", map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
	})

	// Get the next job with available work (respects priority + FIFO and max_agents)
	nextJobWithWork, err := s.jobExecutionService.GetNextJobWithWork(ctx)
	if err != nil {
		debug.Log("Error getting next job with work", map[string]interface{}{
			"agent_id": agent.ID,
			"error":    err.Error(),
		})
		return nil, nil, fmt.Errorf("failed to get next job with work: %w", err)
	}

	if nextJobWithWork == nil {
		debug.Log("No jobs with available work for agent", map[string]interface{}{
			"agent_id": agent.ID,
		})
		return nil, nil, nil // No work available
	}

	// Convert JobExecutionWithWork to JobExecution for compatibility
	nextJob := &nextJobWithWork.JobExecution

	debug.Log("Found pending job for agent", map[string]interface{}{
		"agent_id":         agent.ID,
		"job_execution_id": nextJob.ID,
		"job_priority":     nextJob.Priority,
		"preset_job_name":  nextJob.PresetJobName,
		"hashlist_name":    nextJob.HashlistName,
	})

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
	debug.Log("Syncing hashlist to agent", map[string]interface{}{
		"agent_id":    agent.ID,
		"hashlist_id": nextJob.HashlistID,
	})
	err = s.hashlistSyncService.EnsureHashlistOnAgent(ctx, agent.ID, nextJob.HashlistID)
	if err != nil {
		debug.Log("Failed to sync hashlist to agent", map[string]interface{}{
			"agent_id":    agent.ID,
			"hashlist_id": nextJob.HashlistID,
			"error":       err.Error(),
		})
		return nil, interruptedJobs, fmt.Errorf("failed to sync hashlist to agent: %w", err)
	}

	// Get hashlist to retrieve hash type
	hashlist, err := s.jobExecutionService.hashlistRepo.GetByID(ctx, nextJob.HashlistID)
	if err != nil {
		return nil, interruptedJobs, fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Check if agent has a benchmark for this attack mode and hash type
	benchmark, err := s.jobExecutionService.benchmarkRepo.GetAgentBenchmark(ctx, agent.ID, nextJob.AttackMode, hashlist.HashTypeID)
	needsBenchmark := err != nil || benchmark == nil

	// If recent benchmark check is needed, check if it's still valid
	if !needsBenchmark && benchmark != nil {
		cacheDuration := 168 * time.Hour // Default 7 days
		if setting, err := s.systemSettingsRepo.GetSetting(ctx, "benchmark_cache_duration_hours"); err == nil && setting.Value != nil {
			if hours, err := strconv.Atoi(*setting.Value); err == nil {
				cacheDuration = time.Duration(hours) * time.Hour
			}
		}

		isRecent, err := s.jobExecutionService.benchmarkRepo.IsRecentBenchmark(ctx, agent.ID, nextJob.AttackMode, hashlist.HashTypeID, cacheDuration)
		needsBenchmark = err != nil || !isRecent
	}

	if needsBenchmark {
		debug.Log("Agent needs benchmark before assignment", map[string]interface{}{
			"agent_id":    agent.ID,
			"attack_mode": nextJob.AttackMode,
			"hash_type":   hashlist.HashTypeID,
		})

		// Request benchmark from agent if WebSocket integration is available
		if s.wsIntegration != nil {
			err = s.wsIntegration.RequestAgentBenchmark(ctx, agent.ID, nextJob)
			if err != nil {
				debug.Log("Failed to request benchmark from agent", map[string]interface{}{
					"agent_id": agent.ID,
					"error":    err.Error(),
				})
				return nil, interruptedJobs, fmt.Errorf("failed to request benchmark: %w", err)
			}

			debug.Log("Benchmark requested from agent, deferring job assignment", map[string]interface{}{
				"agent_id": agent.ID,
				"job_id":   nextJob.ID,
			})

			// Return without assigning work - the agent will be available for assignment
			// once the benchmark completes
			return nil, interruptedJobs, nil
		}

		// If no WebSocket integration, we can't request benchmarks
		return nil, interruptedJobs, fmt.Errorf("benchmark required but WebSocket integration not available")
	}

	// Check if this job uses rule splitting and needs initialization
	if nextJob.UsesRuleSplitting {
		// Check if tasks already exist for this job
		taskCount, err := s.jobExecutionService.jobTaskRepo.GetTaskCountByJobExecution(ctx, nextJob.ID)
		if err != nil {
			return nil, interruptedJobs, fmt.Errorf("failed to get task count: %w", err)
		}

		debug.Log("Checking rule splitting initialization", map[string]interface{}{
			"job_id":             nextJob.ID,
			"rule_split_count":   nextJob.RuleSplitCount,
			"existing_tasks":     taskCount,
			"processed_keyspace": nextJob.ProcessedKeyspace,
		})

		if taskCount == 0 {
			// This is the first time, need to initialize rule splitting
			debug.Log("Initializing rule splitting for job", map[string]interface{}{
				"job_id":           nextJob.ID,
				"rule_split_count": nextJob.RuleSplitCount,
			})

			// Let the job execution service handle rule splitting
			err = s.jobExecutionService.InitializeRuleSplitting(ctx, nextJob)
			if err != nil {
				debug.Error("Failed to initialize rule splitting: %v", err)
				return nil, interruptedJobs, fmt.Errorf("failed to initialize rule splitting: %w", err)
			}

			debug.Log("Rule splitting initialized successfully", map[string]interface{}{
				"job_id": nextJob.ID,
			})
		}
	}

	// Calculate the next chunk for this agent
	chunkReq := ChunkCalculationRequest{
		JobExecution:  nextJob,
		Agent:         agent,
		AttackMode:    nextJob.AttackMode,
		HashType:      hashlist.HashTypeID,
		ChunkDuration: 1200, // This should come from settings or preset job
	}

	// Get chunk duration from settings or preset job
	if chunkDuration, err := s.getChunkDuration(ctx, nextJob); err == nil {
		chunkReq.ChunkDuration = chunkDuration
	}

	debug.Log("Calculating chunk for agent", map[string]interface{}{
		"agent_id":       agent.ID,
		"attack_mode":    chunkReq.AttackMode,
		"chunk_duration": chunkReq.ChunkDuration,
	})

	// For rule-split jobs, we need special handling
	var jobTask *models.JobTask
	if nextJob.UsesRuleSplitting {
		// Get the next available rule chunk task
		debug.Log("Getting next rule split task", map[string]interface{}{
			"job_id":   nextJob.ID,
			"agent_id": agent.ID,
		})

		jobTask, err = s.jobExecutionService.GetNextRuleSplitTask(ctx, nextJob, agent)
		if err != nil {
			return nil, interruptedJobs, fmt.Errorf("failed to get rule split task: %w", err)
		}
		if jobTask == nil {
			// No more tasks available for this job
			debug.Log("No more rule split tasks available", map[string]interface{}{
				"job_id": nextJob.ID,
			})
			return nil, interruptedJobs, nil
		}

		debug.Log("Got rule split task", map[string]interface{}{
			"task_id":            jobTask.ID,
			"is_rule_split_task": jobTask.IsRuleSplitTask,
			"rule_chunk_path":    jobTask.RuleChunkPath,
		})
	} else {
		// Regular chunking
		chunkResult, err := s.jobChunkingService.CalculateNextChunk(ctx, chunkReq)
		if err != nil {
			debug.Log("Failed to calculate chunk", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    err.Error(),
			})
			return nil, interruptedJobs, fmt.Errorf("failed to calculate chunk: %w", err)
		}

		debug.Log("Chunk calculated successfully", map[string]interface{}{
			"agent_id":        agent.ID,
			"keyspace_start":  chunkResult.KeyspaceStart,
			"keyspace_end":    chunkResult.KeyspaceEnd,
			"benchmark_speed": chunkResult.BenchmarkSpeed,
		})

		// Create the job task
		jobTask, err = s.jobExecutionService.CreateJobTask(
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
	}

	// Sync any rule chunks if this is a rule split task
	if jobTask.IsRuleSplitTask {
		err = s.hashlistSyncService.SyncJobFiles(ctx, agent.ID, jobTask)
		if err != nil {
			debug.Log("Failed to sync rule chunk to agent", map[string]interface{}{
				"agent_id": agent.ID,
				"task_id":  jobTask.ID,
				"error":    err.Error(),
			})
			// Don't fail the task assignment - the agent will get the file on demand
		}
	}

	// Start the job execution if it's in pending status (handles both initial start and restart after errors)
	if nextJob.Status == models.JobExecutionStatusPending {
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
		"agent_id":         agent.ID,
		"job_task_id":      jobTask.ID,
		"job_execution_id": nextJob.ID,
		"keyspace_start":   jobTask.KeyspaceStart,
		"keyspace_end":     jobTask.KeyspaceEnd,
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

// HandleTaskSuccess handles successful task completion and resets consecutive failure counters
func (s *JobSchedulingService) HandleTaskSuccess(ctx context.Context, taskID uuid.UUID) error {
	// Get the task to find the job execution and agent
	task, err := s.jobExecutionService.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Reset job's consecutive failures
	jobExecution, err := s.jobExecutionService.jobExecRepo.GetByID(ctx, task.JobExecutionID)
	if err == nil && jobExecution.ConsecutiveFailures > 0 {
		err = s.jobExecutionService.jobExecRepo.UpdateConsecutiveFailures(ctx, task.JobExecutionID, 0)
		if err != nil {
			debug.Log("Failed to reset job consecutive failures", map[string]interface{}{
				"job_execution_id": task.JobExecutionID,
				"error":            err.Error(),
			})
		}
	}

	// Reset agent's consecutive failures if assigned
	if task.AgentID != nil {
		agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
		if err == nil && agent.ConsecutiveFailures > 0 {
			err = s.agentRepo.UpdateConsecutiveFailures(ctx, *task.AgentID, 0)
			if err != nil {
				debug.Log("Failed to reset agent consecutive failures", map[string]interface{}{
					"agent_id": *task.AgentID,
					"error":    err.Error(),
				})
			}
		}
	}

	return nil
}

// StartScheduler starts the job scheduler with periodic scheduling
func (s *JobSchedulingService) StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	debug.Log("Job scheduler started", map[string]interface{}{
		"interval": interval,
	})

	// Recover stale jobs on startup
	if err := s.RecoverStaleJobs(ctx); err != nil {
		debug.Log("Failed to recover stale jobs on startup", map[string]interface{}{
			"error": err.Error(),
		})
	}

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
	// Use the enhanced progress tracking method from job execution service
	err := s.jobExecutionService.UpdateTaskProgress(ctx, taskID, progress.KeyspaceProcessed, &progress.HashRate, progress.ProgressPercent)
	if err != nil {
		return fmt.Errorf("failed to update task progress: %w", err)
	}

	// Get the task to find the job execution and agent ID
	task, err := s.jobExecutionService.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
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

	// Store device-specific metrics if available
	if len(progress.DeviceMetrics) > 0 && task.AgentID != nil {
		// Get the job execution to get attack mode
		jobExec, err := s.jobExecutionService.jobExecRepo.GetByID(ctx, task.JobExecutionID)
		if err != nil {
			debug.Log("Failed to get job execution for device metrics", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			attackMode := int(jobExec.AttackMode)
			
			// Store metrics for each device
			for _, device := range progress.DeviceMetrics {
				timestamp := time.Now()
				
				// Store temperature metric
				if device.Temp > 0 {
					tempMetric := &models.AgentPerformanceMetric{
						AgentID:          *task.AgentID,
						MetricType:       models.MetricTypeTemperature,
						Value:            device.Temp,
						Timestamp:        timestamp,
						AggregationLevel: models.AggregationLevelRealtime,
						DeviceID:         &device.DeviceID,
						DeviceName:       &device.DeviceName,
						TaskID:           &taskID,
						AttackMode:       &attackMode,
					}
					if err := s.jobExecutionService.benchmarkRepo.CreateAgentPerformanceMetric(ctx, tempMetric); err != nil {
						debug.Log("Failed to store temperature metric", map[string]interface{}{"error": err.Error()})
					}
				}

				// Store utilization metric
				if device.Util >= 0 {
					utilMetric := &models.AgentPerformanceMetric{
						AgentID:          *task.AgentID,
						MetricType:       models.MetricTypeUtilization,
						Value:            device.Util,
						Timestamp:        timestamp,
						AggregationLevel: models.AggregationLevelRealtime,
						DeviceID:         &device.DeviceID,
						DeviceName:       &device.DeviceName,
						TaskID:           &taskID,
						AttackMode:       &attackMode,
					}
					if err := s.jobExecutionService.benchmarkRepo.CreateAgentPerformanceMetric(ctx, utilMetric); err != nil {
						debug.Log("Failed to store utilization metric", map[string]interface{}{"error": err.Error()})
					}
				}

				// Store fan speed metric (custom metric type)
				if device.FanSpeed >= 0 {
					// Note: Using power_usage as a placeholder for fan speed since it's not in the enum
					fanMetric := &models.AgentPerformanceMetric{
						AgentID:          *task.AgentID,
						MetricType:       models.MetricTypePowerUsage, // TODO: Add MetricTypeFanSpeed to enum
						Value:            device.FanSpeed,
						Timestamp:        timestamp,
						AggregationLevel: models.AggregationLevelRealtime,
						DeviceID:         &device.DeviceID,
						DeviceName:       &device.DeviceName,
						TaskID:           &taskID,
						AttackMode:       &attackMode,
					}
					if err := s.jobExecutionService.benchmarkRepo.CreateAgentPerformanceMetric(ctx, fanMetric); err != nil {
						debug.Log("Failed to store fan speed metric", map[string]interface{}{"error": err.Error()})
					}
				}

				// Store hash rate metric per device
				if device.Speed > 0 {
					hashRateMetric := &models.AgentPerformanceMetric{
						AgentID:          *task.AgentID,
						MetricType:       models.MetricTypeHashRate,
						Value:            float64(device.Speed),
						Timestamp:        timestamp,
						AggregationLevel: models.AggregationLevelRealtime,
						DeviceID:         &device.DeviceID,
						DeviceName:       &device.DeviceName,
						TaskID:           &taskID,
						AttackMode:       &attackMode,
					}
					if err := s.jobExecutionService.benchmarkRepo.CreateAgentPerformanceMetric(ctx, hashRateMetric); err != nil {
						debug.Log("Failed to store hash rate metric", map[string]interface{}{"error": err.Error()})
					}
				}
			}
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

// StopJob stops a running job execution and all its tasks
func (s *JobSchedulingService) StopJob(ctx context.Context, jobExecutionID uuid.UUID, reason string) error {
	// Update job execution status to cancelled
	err := s.jobExecutionService.jobExecRepo.UpdateStatus(ctx, jobExecutionID, models.JobExecutionStatusCancelled)
	if err != nil {
		return fmt.Errorf("failed to update job execution status: %w", err)
	}

	// Cancel all running tasks
	tasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	for _, task := range tasks {
		if task.Status == models.JobTaskStatusRunning || task.Status == models.JobTaskStatusAssigned {
			err = s.jobExecutionService.jobTaskRepo.CancelTask(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to cancel task", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
			}
		}
	}

	debug.Log("Job execution stopped", map[string]interface{}{
		"job_execution_id": jobExecutionID,
		"reason":           reason,
	})

	return nil
}

// RecoverStaleJobs recovers jobs that were assigned but not completed when server restarts
func (s *JobSchedulingService) RecoverStaleJobs(ctx context.Context) error {
	debug.Log("Starting stale job recovery", nil)

	// Get all tasks that are in 'assigned' or 'running' state
	staleStatuses := []string{"assigned", "running"}
	staleTasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByStatuses(ctx, staleStatuses)
	if err != nil {
		return fmt.Errorf("failed to get stale tasks: %w", err)
	}

	if len(staleTasks) == 0 {
		debug.Log("No stale tasks found", nil)
		return nil
	}

	debug.Log("Found stale tasks to recover", map[string]interface{}{
		"task_count": len(staleTasks),
	})

	// Reset each stale task back to pending
	for _, task := range staleTasks {
		// Check if the agent is currently connected
		if task.AgentID == nil {
			// Task was never assigned to an agent, just reset it
			err = s.jobExecutionService.jobTaskRepo.ResetTaskForRetry(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to reset unassigned stale task", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
			}
			continue
		}

		agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
		if err != nil {
			debug.Log("Failed to get agent for stale task", map[string]interface{}{
				"task_id":  task.ID,
				"agent_id": task.AgentID,
				"error":    err.Error(),
			})
			continue
		}

		// If agent is active and connected, we'll wait for it to report progress
		if agent.Status == "active" {
			// Check last checkpoint time
			if task.LastCheckpoint != nil {
				timeSinceCheckpoint := time.Since(*task.LastCheckpoint)
				// If checkpoint is recent (within 5 minutes), assume task is still running
				if timeSinceCheckpoint < 5*time.Minute {
					debug.Log("Task has recent checkpoint, assuming still running", map[string]interface{}{
						"task_id":               task.ID,
						"time_since_checkpoint": timeSinceCheckpoint,
					})
					continue
				}
			}
		}

		// Reset task to pending for reassignment
		err = s.jobExecutionService.jobTaskRepo.ResetTaskForRetry(ctx, task.ID)
		if err != nil {
			debug.Log("Failed to reset stale task", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			continue
		}

		debug.Log("Reset stale task to pending", map[string]interface{}{
			"task_id":          task.ID,
			"job_execution_id": task.JobExecutionID,
			"agent_id":         task.AgentID,
		})

		// Also update the job execution status if needed
		jobExec, err := s.jobExecutionService.jobExecRepo.GetByID(ctx, task.JobExecutionID)
		if err == nil && jobExec.Status == models.JobExecutionStatusRunning {
			// Check if there are any other running tasks for this job
			activeTasks, err := s.jobExecutionService.jobTaskRepo.GetActiveTasksCount(ctx, task.JobExecutionID)
			if err == nil && activeTasks == 0 {
				// No active tasks, reset job to pending
				err = s.jobExecutionService.jobExecRepo.UpdateStatus(ctx, task.JobExecutionID, models.JobExecutionStatusPending)
				if err != nil {
					debug.Log("Failed to reset job execution status", map[string]interface{}{
						"job_execution_id": task.JobExecutionID,
						"error":            err.Error(),
					})
				}
			}
		}
	}

	debug.Log("Stale job recovery completed", map[string]interface{}{
		"total_tasks_recovered": len(staleTasks),
	})

	return nil
}
