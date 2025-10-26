package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	SendJobStop(ctx context.Context, taskID uuid.UUID, reason string) error
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
		
		// Check for high-priority jobs that can interrupt running jobs
		// This only happens when NO agents are available
		interruptedJobID, err := s.checkAndInterruptForHighPriority(ctx)
		if err != nil {
			debug.Log("Error checking for high-priority interruptions", map[string]interface{}{
				"error": err.Error(),
			})
		} else if interruptedJobID != nil {
			debug.Log("Interrupted job for high-priority override", map[string]interface{}{
				"interrupted_job_id": *interruptedJobID,
			})
			result.InterruptedJobs = append(result.InterruptedJobs, *interruptedJobID)
			
			// Re-get available agents after interruption
			// The interrupted task's agent should now be available
			availableAgents, err = s.jobExecutionService.GetAvailableAgents(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get available agents after interruption: %w", err)
			}
			
			debug.Log("Re-checked available agents after interruption", map[string]interface{}{
				"agent_count": len(availableAgents),
			})
		}
		
		// If still no agents available after interruption check, return
		if len(availableAgents) == 0 {
			return result, nil
		}
	}

	debug.Log("Found available agents", map[string]interface{}{
		"agent_count": len(availableAgents),
	})

	// Process each available agent
	for _, agent := range availableAgents {
		taskAssigned, interruptedJobs, err := s.assignWorkToAgent(ctx, &agent)
		if err != nil {
			assignErr := fmt.Errorf("failed to assign work to agent %d: %w", agent.ID, err)
			result.Errors = append(result.Errors, assignErr)
			debug.Error("Failed to assign work to agent: %v", assignErr)
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

	// Check if agent is marked as busy (has a running task)
	if agent.Metadata != nil {
		if busyStatus, exists := agent.Metadata["busy_status"]; exists && busyStatus == "true" {
			// Validate that the task actually exists and is assigned to this agent
			if taskIDStr, exists := agent.Metadata["current_task_id"]; exists && taskIDStr != "" {
				taskUUID, err := uuid.Parse(taskIDStr)
				if err != nil {
					// Invalid task ID, clear stale busy status
					debug.Log("Clearing stale busy status with invalid task ID", map[string]interface{}{
						"agent_id":     agent.ID,
						"invalid_task": taskIDStr,
						"error":        err.Error(),
					})
					agent.Metadata["busy_status"] = "false"
					delete(agent.Metadata, "current_task_id")
					delete(agent.Metadata, "current_job_id")
					s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata)
					// Continue to assign work
				} else {
					// Valid UUID, check if task exists and is actually assigned to this agent
					task, err := s.jobExecutionService.jobTaskRepo.GetByID(ctx, taskUUID)
					if err != nil || task == nil {
						// Task doesn't exist, clear stale busy status
						debug.Log("Clearing stale busy status for non-existent task", map[string]interface{}{
							"agent_id":      agent.ID,
							"stale_task_id": taskIDStr,
						})
						agent.Metadata["busy_status"] = "false"
						delete(agent.Metadata, "current_task_id")
						delete(agent.Metadata, "current_job_id")
						s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata)
						// Continue to assign work
					} else if task.AgentID == nil || *task.AgentID != agent.ID {
						// Task exists but not assigned to this agent
						debug.Log("Clearing stale busy status for unassigned task", map[string]interface{}{
							"agent_id":         agent.ID,
							"task_id":          taskIDStr,
							"task_assigned_to": task.AgentID,
						})
						agent.Metadata["busy_status"] = "false"
						delete(agent.Metadata, "current_task_id")
						delete(agent.Metadata, "current_job_id")
						s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata)
						// Continue to assign work
					} else if task.Status != models.JobTaskStatusRunning && task.Status != models.JobTaskStatusAssigned {
						// Task is not in a running state
						debug.Log("Clearing stale busy status for non-running task", map[string]interface{}{
							"agent_id":    agent.ID,
							"task_id":     taskIDStr,
							"task_status": task.Status,
						})
						agent.Metadata["busy_status"] = "false"
						delete(agent.Metadata, "current_task_id")
						delete(agent.Metadata, "current_job_id")
						s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata)
						// Continue to assign work
					} else {
						// Valid busy status, agent is actually busy
						debug.Log("Agent is busy with a running task", map[string]interface{}{
							"agent_id":   agent.ID,
							"agent_name": agent.Name,
							"task_id":    taskIDStr,
						})
						return nil, nil, nil // Agent is busy, skip assignment
					}
				}
			} else {
				// No task ID but marked as busy, clear stale busy status
				debug.Log("Clearing stale busy status with no task ID", map[string]interface{}{
					"agent_id": agent.ID,
				})
				agent.Metadata["busy_status"] = "false"
				delete(agent.Metadata, "current_task_id")
				delete(agent.Metadata, "current_job_id")
				s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata)
				// Continue to assign work
			}
		}
	}

	// Check if agent has any tasks in reconnect_pending status
	reconnectPendingTasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByAgentAndStatus(ctx, agent.ID, models.JobTaskStatusReconnectPending)
	if err != nil {
		debug.Log("Failed to check for reconnect_pending tasks", map[string]interface{}{
			"agent_id": agent.ID,
			"error":    err.Error(),
		})
	} else if len(reconnectPendingTasks) > 0 {
		// Check if agent is actually busy (has reported a running task)
		isBusy := false
		if agent.Metadata != nil {
			if busyStatus, ok := agent.Metadata["busy_status"]; ok && busyStatus == "true" {
				isBusy = true
			}
		}
		
		if isBusy {
			debug.Log("Agent has reconnect_pending tasks and is busy, waiting for recovery", map[string]interface{}{
				"agent_id":    agent.ID,
				"task_count":  len(reconnectPendingTasks),
				"task_ids":    reconnectPendingTasks,
			})
			return nil, nil, nil // Agent is still running the task
		} else {
			debug.Log("Agent has reconnect_pending tasks but is not busy, these should have been reset", map[string]interface{}{
				"agent_id":    agent.ID,
				"task_count":  len(reconnectPendingTasks),
			})
			// Continue with assignment - the tasks should have been reset already
		}
	}

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
		"job_name":         nextJob.Name,
		"hashlist_id":      nextJob.HashlistID,
	})

	// PREVENTION: Check if hashlist is fully cracked before creating new tasks
	hashlist, err := s.jobExecutionService.hashlistRepo.GetByID(ctx, nextJob.HashlistID)
	if err != nil {
		debug.Error("Failed to get hashlist %d for completion check: %v", nextJob.HashlistID, err)
		// Continue anyway - this is a safety check
	} else if hashlist.CrackedHashes >= hashlist.TotalHashes {
		debug.Warning("Hashlist %d is fully cracked (%d/%d), skipping task assignment for job %s",
			nextJob.HashlistID, hashlist.CrackedHashes, hashlist.TotalHashes, nextJob.ID)
		// Don't create tasks for fully cracked hashlists
		// The hashlist completion handler should clean this up
		return nil, nil, nil
	}

	// Note: Interruption logic has been moved to main ScheduleJobs method
	// and only runs when no agents are available
	var interruptedJobs []uuid.UUID

	// Check for stale benchmark requests (timeout after 5 minutes)
	if agent.Metadata != nil {
		if requestedAt, exists := agent.Metadata["benchmark_requested_at"]; exists {
			if parsedTime, err := time.Parse(time.RFC3339, requestedAt); err == nil {
				if time.Since(parsedTime) > 5*time.Minute {
					debug.Warning("Benchmark request for agent %d timed out after 5 minutes, clearing and retrying", agent.ID)
					delete(agent.Metadata, "pending_benchmark_job")
					delete(agent.Metadata, "benchmark_requested_at")
					s.agentRepo.Update(ctx, agent)
					// Will retry benchmark below if needed
				}
			}
		}
	}

	// Check if this job needs a forced benchmark before first task assignment
	if !nextJob.IsAccurateKeyspace {
		// Check if any tasks have been created for this job yet
		taskCount, err := s.jobExecutionService.jobTaskRepo.GetTaskCountForJob(ctx, nextJob.ID)
		if err != nil {
			debug.Warning("Failed to check task count for job %s: %v", nextJob.ID, err)
		} else if taskCount == 0 {
			// This is the first task assignment - force a benchmark
			debug.Info("Job %s needs forced benchmark before first task assignment", nextJob.ID)

			// Check if benchmark is already pending/in-progress for this job
			if agent.Metadata != nil {
				if pendingBench, exists := agent.Metadata["pending_benchmark_job"]; exists && pendingBench == nextJob.ID.String() {
					debug.Info("Benchmark already pending for job %s on agent %d, waiting...", nextJob.ID, agent.ID)
					return nil, nil, nil // Benchmark in progress, don't assign yet
				}
			}

			// Mark agent as having pending benchmark for this job
			if agent.Metadata == nil {
				agent.Metadata = make(map[string]string)
			}
			agent.Metadata["pending_benchmark_job"] = nextJob.ID.String()
			agent.Metadata["benchmark_requested_at"] = time.Now().Format(time.RFC3339)
			err = s.agentRepo.Update(ctx, agent)
			if err != nil {
				debug.Warning("Failed to update agent metadata for benchmark: %v", err)
			}

			// Send benchmark request to agent
			err = s.wsIntegration.RequestAgentBenchmark(ctx, agent.ID, nextJob)
			if err != nil {
				// Failed to send benchmark - clear metadata and fall back to estimation
				debug.Error("Failed to send benchmark request for job %s to agent %d: %v", nextJob.ID, agent.ID, err)
				if agent.Metadata != nil {
					delete(agent.Metadata, "pending_benchmark_job")
					delete(agent.Metadata, "benchmark_requested_at")
					s.agentRepo.Update(ctx, agent)
				}
				// Continue with task assignment using estimated keyspace
			} else {
				debug.Info("Sent forced benchmark request for job %s to agent %d", nextJob.ID, agent.ID)
				return nil, nil, nil // Wait for benchmark to complete before assigning task
			}
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

	// Hashlist was already retrieved in the prevention check above, so reuse it
	// If there was an error getting it before, try again here
	if hashlist == nil {
		hashlist, err = s.jobExecutionService.hashlistRepo.GetByID(ctx, nextJob.HashlistID)
		if err != nil {
			return nil, interruptedJobs, fmt.Errorf("failed to get hashlist: %w", err)
		}
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
			"agent_id":         agent.ID,
			"attack_mode":      nextJob.AttackMode,
			"hash_type":        hashlist.HashTypeID,
			"binary_version_id": nextJob.BinaryVersionID,
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

	// For rule splitting jobs, first check if there are any existing tasks that need assignment
	if nextJob.UsesRuleSplitting {
		// Check for tasks that need to be assigned (error retry, pending, or unassigned)
		// Priority order:
		// 1. Tasks in error state with retry_count < 3
		// 2. Tasks that were returned to pending (agent crashed)
		// 3. Unassigned pending tasks (for backward compatibility)
		
		// First check for error tasks that can be retried
		errorTask, err := s.jobExecutionService.jobTaskRepo.GetRetriableErrorTask(ctx, nextJob.ID, 3)
		if err == nil && errorTask != nil {
			debug.Log("Found error task to retry", map[string]interface{}{
				"task_id":     errorTask.ID,
				"retry_count": errorTask.RetryCount,
				"agent_id":    agent.ID,
			})
			
			// Assign the task to this agent
			errorTask.AgentID = &agent.ID
			errorTask.Status = models.JobTaskStatusPending
			errorTask.RetryCount++
			now := time.Now()
			errorTask.AssignedAt = &now
			errorTask.UpdatedAt = time.Now()
			errorTask.ErrorMessage = nil
			
			if err := s.jobExecutionService.jobTaskRepo.Update(ctx, errorTask); err != nil {
				return nil, interruptedJobs, fmt.Errorf("failed to update error task: %w", err)
			}
			
			return errorTask, interruptedJobs, nil
		}
		
		// Check for tasks returned to pending (stale assignments)
		staleTask, err := s.jobExecutionService.jobTaskRepo.GetStalePendingTask(ctx, nextJob.ID, 5*time.Minute)
		if err == nil && staleTask != nil {
			debug.Log("Found stale pending task to reassign", map[string]interface{}{
				"task_id":         staleTask.ID,
				"previous_agent":  staleTask.AgentID,
				"new_agent":       agent.ID,
				"last_checkpoint": staleTask.LastCheckpoint,
			})
			
			// Reassign the task
			staleTask.AgentID = &agent.ID
			now := time.Now()
			staleTask.AssignedAt = &now
			staleTask.UpdatedAt = time.Now()
			
			if err := s.jobExecutionService.jobTaskRepo.Update(ctx, staleTask); err != nil {
				return nil, interruptedJobs, fmt.Errorf("failed to update stale task: %w", err)
			}
			
			return staleTask, interruptedJobs, nil
		}
		
		// Check for any unassigned pending tasks (backward compatibility)
		unassignedTask, err := s.jobExecutionService.jobTaskRepo.GetUnassignedPendingTask(ctx, nextJob.ID)
		if err == nil && unassignedTask != nil {
			debug.Log("Found unassigned pending task", map[string]interface{}{
				"task_id":  unassignedTask.ID,
				"agent_id": agent.ID,
			})
			
			// Assign the task
			unassignedTask.AgentID = &agent.ID
			now := time.Now()
			unassignedTask.AssignedAt = &now
			unassignedTask.UpdatedAt = time.Now()
			
			if err := s.jobExecutionService.jobTaskRepo.Update(ctx, unassignedTask); err != nil {
				return nil, interruptedJobs, fmt.Errorf("failed to update unassigned task: %w", err)
			}
			
			return unassignedTask, interruptedJobs, nil
		}
	}

	// Check if this is the first dispatch for a job with rules (dynamic rule splitting determination)
	if nextJob.AttackMode == models.AttackModeStraight && 
		nextJob.MultiplicationFactor > 1 && 
		!nextJob.UsesRuleSplitting &&
		benchmark != nil && benchmark.Speed > 0 {
		
		// Only do this check for the first dispatch
		if nextJob.DispatchedKeyspace == 0 {
			// Calculate if the entire job can be done within chunk duration
			effectiveKeyspace := int64(0)
			if nextJob.EffectiveKeyspace != nil {
				effectiveKeyspace = *nextJob.EffectiveKeyspace
			}
			
			// Get chunk duration from settings or preset job
			chunkDuration := 1200 // Default 20 minutes
			if duration, err := s.getChunkDuration(ctx, nextJob); err == nil {
				chunkDuration = duration
			}
			
			// Get fluctuation settings
			fluctuationSetting, _ := s.systemSettingsRepo.GetSetting(ctx, "chunk_fluctuation_percentage")
			fluctuationPercent := 20 // Default 20%
			if fluctuationSetting != nil && fluctuationSetting.Value != nil {
				if parsed, err := strconv.Atoi(*fluctuationSetting.Value); err == nil {
					fluctuationPercent = parsed
				}
			}
			
			// Calculate max allowed duration (chunk duration + fluctuation)
			maxDuration := float64(chunkDuration) * (1.0 + float64(fluctuationPercent)/100.0)
			
			// Estimate time to complete based on benchmark
			estimatedTime := float64(effectiveKeyspace) / float64(benchmark.Speed)
			
			// If job would take longer than max duration, enable rule splitting
			if estimatedTime > maxDuration {
				nextJob.UsesRuleSplitting = true
				nextJob.RuleSplitCount = 0  // Start at 0, will increment as chunks are created
				// Update the job in database
				err = s.jobExecutionService.UpdateRuleSplitting(ctx, nextJob.ID, true)
				if err != nil {
					debug.Log("Failed to update rule splitting flag", map[string]interface{}{
						"job_id": nextJob.ID,
						"error": err.Error(),
					})
				}
				
				// Update rule split count to 0
				err = s.jobExecutionService.jobExecRepo.UpdateKeyspaceInfo(ctx, nextJob)
				if err != nil {
					debug.Log("Failed to reset rule split count", map[string]interface{}{
						"job_id": nextJob.ID,
						"error": err.Error(),
					})
				}
				
				debug.Log("Dynamically enabled rule splitting", map[string]interface{}{
					"job_id": nextJob.ID,
					"effective_keyspace": effectiveKeyspace,
					"benchmark_speed": benchmark.Speed,
					"estimated_time": estimatedTime,
					"max_duration": maxDuration,
					"chunk_duration": chunkDuration,
					"fluctuation_percent": fluctuationPercent,
					"rule_split_count": 0,
				})
			} else {
				debug.Log("Job can be completed in single chunk", map[string]interface{}{
					"job_id": nextJob.ID,
					"effective_keyspace": effectiveKeyspace,
					"benchmark_speed": benchmark.Speed,
					"estimated_time": estimatedTime,
					"max_duration": maxDuration,
				})
			}
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
		// First check if there are any pending tasks for this job
		pendingTasks, err := s.jobExecutionService.jobTaskRepo.GetPendingTasksByJobExecution(ctx, nextJob.ID)
		if err != nil {
			debug.Log("Failed to get pending tasks", map[string]interface{}{
				"job_id": nextJob.ID,
				"error":  err.Error(),
			})
		}
		
		if len(pendingTasks) > 0 {
			// Assign the first pending task to this agent
			pendingTask := &pendingTasks[0]
			debug.Log("Assigning existing pending task to agent", map[string]interface{}{
				"job_id":       nextJob.ID,
				"agent_id":     agent.ID,
				"task_id":      pendingTask.ID,
				"chunk_number": pendingTask.ChunkNumber,
				"rule_start":   pendingTask.RuleStartIndex,
				"rule_end":     pendingTask.RuleEndIndex,
			})
			
			// Update the task with the new agent assignment
			pendingTask.AgentID = &agent.ID
			pendingTask.Status = models.JobTaskStatusAssigned
			now := time.Now()
			pendingTask.AssignedAt = &now
			
			// Update in database
			err = s.jobExecutionService.jobTaskRepo.AssignTaskToAgent(ctx, pendingTask.ID, agent.ID)
			if err != nil {
				return nil, interruptedJobs, fmt.Errorf("failed to assign pending task to agent: %w", err)
			}
			
			// Update dispatched keyspace for the job
			if pendingTask.EffectiveKeyspaceStart != nil && pendingTask.EffectiveKeyspaceEnd != nil {
				dispatchedKeyspace := *pendingTask.EffectiveKeyspaceEnd - *pendingTask.EffectiveKeyspaceStart
				err = s.jobExecutionService.jobExecRepo.IncrementDispatchedKeyspace(ctx, nextJob.ID, dispatchedKeyspace)
				if err != nil {
					debug.Error("Failed to update dispatched keyspace: %v", err)
				}
			}
			
			jobTask = pendingTask
		} else {
			// No pending tasks, create a new chunk
			debug.Log("No pending tasks found, creating new dynamic rule chunk", map[string]interface{}{
				"job_id":   nextJob.ID,
				"agent_id": agent.ID,
			})

			// Get the next rule index by checking existing tasks
			maxRuleEnd, err := s.jobExecutionService.jobTaskRepo.GetMaxRuleEndIndex(ctx, nextJob.ID)
			nextRuleStart := 0
			if maxRuleEnd != nil {
				nextRuleStart = *maxRuleEnd
			}

		// Check if all rules have been dispatched
		totalRules := 0
		if nextJob.RuleSplitCount > 0 {
			// Get total rules from job metadata
			// The job_executions table contains all needed information
			
			// Get the rule file to count total rules
			if len(nextJob.RuleIDs) > 0 {
				rulePath, err := s.jobExecutionService.resolveRulePath(ctx, nextJob.RuleIDs[0])
				if err != nil {
					return nil, interruptedJobs, fmt.Errorf("failed to resolve rule path: %w", err)
				}
				totalRules, err = s.jobExecutionService.ruleSplitManager.CountRules(ctx, rulePath)
				if err != nil {
					return nil, interruptedJobs, fmt.Errorf("failed to count rules: %w", err)
				}
			}
		}

		if nextRuleStart >= totalRules {
			debug.Log("All rules have been dispatched", map[string]interface{}{
				"job_id":          nextJob.ID,
				"total_rules":     totalRules,
				"next_rule_start": nextRuleStart,
			})
			
			// Check if job should be completed
			err = s.ProcessJobCompletion(ctx, nextJob.ID)
			if err != nil {
				debug.Log("Failed to process job completion", map[string]interface{}{
					"job_id": nextJob.ID,
					"error":  err.Error(),
				})
			}
			
			return nil, interruptedJobs, nil
		}

		// Calculate rules for this specific agent based on its benchmark
		// For rule splits: effective speed = base_keyspace_per_second / chunk_duration * rules_per_chunk
		baseKeyspace := int64(0)
		if nextJob.BaseKeyspace != nil {
			baseKeyspace = *nextJob.BaseKeyspace
		}

		// Calculate how many rules this agent can process in the chunk duration
		// Get benchmark speed for this agent
		benchmarkSpeed, err := s.jobChunkingService.GetOrEstimateBenchmark(ctx, agent.ID, nextJob.AttackMode, hashlist.HashTypeID)
		if err != nil {
			debug.Log("Failed to get benchmark, using default", map[string]interface{}{
				"error": err.Error(),
			})
			benchmarkSpeed = 1000000 // Default 1M H/s
		}

		// rulesPerSecond = benchmarkSpeed / baseKeyspace (how many complete wordlist passes per second)
		// rulesPerChunk = rulesPerSecond * chunkDuration
		rulesPerChunk := 100 // Default if calculation fails
		if baseKeyspace > 0 && benchmarkSpeed > 0 {
			rulesPerSecond := float64(benchmarkSpeed) / float64(baseKeyspace)
			rulesPerChunk = int(rulesPerSecond * float64(chunkReq.ChunkDuration))
			if rulesPerChunk < 1 {
				rulesPerChunk = 1 // At least one rule per chunk
			}
		}

		// Apply fluctuation logic to avoid tiny final chunks
		fluctuationSetting, _ := s.systemSettingsRepo.GetSetting(ctx, "chunk_fluctuation_percentage")
		fluctuationPercent := 20 // Default 20%
		if fluctuationSetting != nil && fluctuationSetting.Value != nil {
			if parsed, err := strconv.Atoi(*fluctuationSetting.Value); err == nil {
				fluctuationPercent = parsed
			}
		}

		fluctuationThreshold := int(float64(rulesPerChunk) * float64(fluctuationPercent) / 100.0)
		nextRuleEnd := nextRuleStart + rulesPerChunk

		if nextRuleEnd >= totalRules {
			nextRuleEnd = totalRules
		} else {
			// Check if remaining rules would be too small
			remainingAfterChunk := totalRules - nextRuleEnd
			if remainingAfterChunk <= fluctuationThreshold {
				// Merge the final small chunk
				nextRuleEnd = totalRules
				debug.Log("Merging final rule chunk to avoid small remainder", map[string]interface{}{
					"normal_chunk_size":    rulesPerChunk,
					"remaining_rules":      remainingAfterChunk,
					"threshold":            fluctuationThreshold,
					"merged_chunk_size":    nextRuleEnd - nextRuleStart,
					"percent_over_normal":  float64(nextRuleEnd-nextRuleStart-rulesPerChunk) / float64(rulesPerChunk) * 100,
				})
			}
		}

		// Create rule chunk file on-demand
		// Get the rule path from the job execution (which has all needed data)
		var rulePath string
		if len(nextJob.RuleIDs) > 0 {
			rulePath, _ = s.jobExecutionService.resolveRulePath(ctx, nextJob.RuleIDs[0])
		}
		chunk, err := s.jobExecutionService.ruleSplitManager.CreateSingleRuleChunk(
			ctx, nextJob.ID, rulePath, nextRuleStart, nextRuleEnd-nextRuleStart)
		if err != nil {
			return nil, interruptedJobs, fmt.Errorf("failed to create rule chunk: %w", err)
		}

		// Get next chunk number
		debug.Log("Getting next chunk number", map[string]interface{}{
			"job_id": nextJob.ID,
		})
		chunkNumber, err := s.jobExecutionService.jobTaskRepo.GetNextChunkNumber(ctx, nextJob.ID)
		if err != nil {
			debug.Error("Failed to get next chunk number: %v", err)
			fmt.Printf("ERROR in assignWorkToAgent: Failed to get next chunk number for job %s: %v\n", nextJob.ID, err)
			return nil, interruptedJobs, fmt.Errorf("failed to get next chunk number: %w", err)
		}
		debug.Log("Got chunk number", map[string]interface{}{
			"job_id":       nextJob.ID,
			"chunk_number": chunkNumber,
		})

		// Build attack command
		// For custom jobs, pass nil for presetJob since all data is in nextJob
		debug.Log("Building attack command", map[string]interface{}{
			"job_id":            nextJob.ID,
			"binary_version_id": nextJob.BinaryVersionID,
			"attack_mode":       nextJob.AttackMode,
			"hash_type":         nextJob.HashType,
			"wordlist_ids":      nextJob.WordlistIDs,
			"rule_ids":          nextJob.RuleIDs,
		})
		attackCmd, err := s.jobExecutionService.buildAttackCommand(ctx, nil, nextJob)
		if err != nil {
			debug.Error("Failed to build attack command: %v", err)
			fmt.Printf("ERROR in assignWorkToAgent: Failed to build attack command for job %s: %v\n", nextJob.ID, err)
			return nil, interruptedJobs, fmt.Errorf("failed to build attack command: %w", err)
		}
		cmdPreview := attackCmd
		if len(attackCmd) > 100 {
			cmdPreview = attackCmd[:100] + "..."
		}
		debug.Log("Built attack command", map[string]interface{}{
			"job_id":      nextJob.ID,
			"cmd_preview": cmdPreview,
		})
		// Replace rule file with chunk path
		attackCmd = strings.Replace(attackCmd, rulePath, chunk.Path, 1)

		// Calculate effective keyspace for this chunk using previous chunks' ACTUAL sizes
		effectiveKeyspaceStart := int64(0)

		// Get cumulative actual keyspace from all previous chunks
		previousChunksActual, err := s.jobExecutionService.GetPreviousChunksActualKeyspace(ctx, nextJob.ID, chunkNumber)
		if err == nil && previousChunksActual > 0 {
			effectiveKeyspaceStart = previousChunksActual
		} else {
			if err != nil {
				debug.Error("Failed to get previous chunks' actual keyspace: %v", err)
			}
			// Fall back to estimated based on base keyspace
			effectiveKeyspaceStart = baseKeyspace * int64(nextRuleStart)
		}

		// For end, use estimated chunk size (will be corrected when hashcat reports actual)
		rulesInChunk := chunk.RuleCount
		estimatedChunkKeyspace := baseKeyspace * int64(rulesInChunk)
		effectiveKeyspaceEnd := effectiveKeyspaceStart + estimatedChunkKeyspace

		debug.Log("Calculated effective keyspace for new chunk", map[string]interface{}{
			"job_id":               nextJob.ID,
			"chunk_number":         chunkNumber,
			"rules_in_chunk":       rulesInChunk,
			"effective_start":      effectiveKeyspaceStart,
			"effective_end":        effectiveKeyspaceEnd,
			"estimated_chunk_size": estimatedChunkKeyspace,
		})

		// Create task
		task := &models.JobTask{
			JobExecutionID:         nextJob.ID,
			AgentID:                &agent.ID,
			Status:                 models.JobTaskStatusPending,
			Priority:               nextJob.Priority,
			AttackCmd:              attackCmd,
			KeyspaceStart:          0,
			KeyspaceEnd:            baseKeyspace,
			KeyspaceProcessed:      0,
			EffectiveKeyspaceStart: &effectiveKeyspaceStart,
			EffectiveKeyspaceEnd:   &effectiveKeyspaceEnd,
			RuleStartIndex:         &chunk.StartIndex,
			RuleEndIndex:           &chunk.EndIndex,
			RuleChunkPath:          &chunk.Path,
			IsRuleSplitTask:        true,
			ChunkNumber:            chunkNumber,
			ChunkDuration:          chunkReq.ChunkDuration,
			BenchmarkSpeed:         &benchmarkSpeed,
		}

		// Save task
		debug.Log("Creating job task", map[string]interface{}{
			"job_id":          nextJob.ID,
			"agent_id":        agent.ID,
			"chunk_number":    chunkNumber,
			"keyspace_start":  task.KeyspaceStart,
			"keyspace_end":    task.KeyspaceEnd,
			"chunk_duration":  task.ChunkDuration,
			"benchmark_speed": benchmarkSpeed,
		})
		err = s.jobExecutionService.jobTaskRepo.Create(ctx, task)
		if err != nil {
			debug.Error("Failed to create job task: %v", err)
			fmt.Printf("ERROR in assignWorkToAgent: Failed to create task for job %s: %v\n", nextJob.ID, err)
			return nil, interruptedJobs, fmt.Errorf("failed to create task: %w", err)
		}
		debug.Log("Successfully created job task", map[string]interface{}{
			"task_id": task.ID,
			"job_id":  nextJob.ID,
		})

		// Update dispatched keyspace
		// For rule splitting, we need to account for the number of rules in this chunk
		dispatchedKeyspace := baseKeyspace * int64(chunk.RuleCount)
		err = s.jobExecutionService.jobExecRepo.IncrementDispatchedKeyspace(ctx, nextJob.ID, dispatchedKeyspace)
		if err != nil {
			debug.Error("Failed to update dispatched keyspace: %v", err)
		}
		
		// Update rule_split_count to reflect actual chunks created
		actualChunksCreated := chunkNumber
		nextJob.RuleSplitCount = actualChunksCreated
		err = s.jobExecutionService.jobExecRepo.UpdateKeyspaceInfo(ctx, nextJob)
		if err != nil {
			debug.Error("Failed to update rule split count: %v", err)
		}
		
		debug.Log("Updated dispatched keyspace and rule split count", map[string]interface{}{
			"job_id":              nextJob.ID,
			"base_keyspace":       baseKeyspace,
			"rules_in_chunk":      chunk.RuleCount,
			"dispatched_keyspace": dispatchedKeyspace,
			"rule_split_count":    actualChunksCreated,
		})

		jobTask = task

		debug.Log("Created dynamic rule chunk task", map[string]interface{}{
			"task_id":         task.ID,
			"chunk_number":    chunkNumber,
			"rule_start":      chunk.StartIndex,
			"rule_end":        chunk.EndIndex,
			"rules_in_chunk":  chunk.RuleCount,
			"chunk_path":      chunk.Path,
		})
		}  // End of else block for "No pending tasks"
	} else {
		// Regular chunking
		chunkResult, err := s.jobChunkingService.CalculateNextChunk(ctx, chunkReq)
		if err != nil {
			// Special handling for "no remaining keyspace" - not an error, just done
			if strings.Contains(err.Error(), "no remaining keyspace") {
				debug.Log("All keyspace has been dispatched for job", map[string]interface{}{
					"job_id": nextJob.ID,
					"total_keyspace": nextJob.TotalKeyspace,
				})

				// Let ProcessJobCompletion handle the completion check
				err = s.ProcessJobCompletion(ctx, nextJob.ID)
				if err != nil {
					debug.Log("Failed to process job completion", map[string]interface{}{
						"job_id": nextJob.ID,
						"error":  err.Error(),
					})
				}

				return nil, interruptedJobs, nil // Return success, no task to create
			}

			// All other errors are actual failures
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
			"benchmark_value": func() int64 {
				if chunkResult.BenchmarkSpeed != nil {
					return *chunkResult.BenchmarkSpeed
				}
				return 0
			}(),
			"chunk_duration": chunkReq.ChunkDuration,
			"chunk_size":     chunkResult.KeyspaceEnd - chunkResult.KeyspaceStart,
		})

		// Create the job task
		jobTask, err = s.jobExecutionService.CreateJobTask(
			ctx,
			nextJob,
			agent,
			chunkResult.KeyspaceStart,
			chunkResult.KeyspaceEnd,
			chunkResult.BenchmarkSpeed,
			chunkReq.ChunkDuration,
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
	// First try to get from job execution itself
	if jobExecution.ChunkSizeSeconds > 0 {
		return jobExecution.ChunkSizeSeconds, nil
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

	// Cleanup ticker runs every 5 minutes
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	debug.Log("Job scheduler started", map[string]interface{}{
		"interval":         interval,
		"cleanup_interval": "5m",
	})

	// Recover stale jobs on startup
	if err := s.RecoverStaleJobs(ctx); err != nil {
		debug.Log("Failed to recover stale jobs on startup", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Run cleanup on startup
	if err := s.CleanupStaleAgentStatus(ctx); err != nil {
		debug.Log("Failed to cleanup stale agent status on startup", map[string]interface{}{
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
		case <-cleanupTicker.C:
			// Run periodic cleanup of stale agent status
			if err := s.CleanupStaleAgentStatus(ctx); err != nil {
				debug.Log("Failed to cleanup stale agent status", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// checkAndInterruptForHighPriority checks if there are high-priority jobs waiting
// that should interrupt lower priority running jobs. This only runs when no agents are available.
func (s *JobSchedulingService) checkAndInterruptForHighPriority(ctx context.Context) (*uuid.UUID, error) {
	// Check if interruption is enabled
	interruptionSetting, err := s.systemSettingsRepo.GetSetting(ctx, "job_interruption_enabled")
	if err != nil {
		return nil, fmt.Errorf("failed to get interruption setting: %w", err)
	}

	if interruptionSetting.Value == nil || *interruptionSetting.Value != "true" {
		return nil, nil // Interruption disabled
	}

	// Get the highest priority pending job with allow_high_priority_override
	highPriorityJobs, err := s.jobExecutionService.jobExecRepo.GetPendingJobsWithHighPriorityOverride(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get high-priority jobs: %w", err)
	}

	if len(highPriorityJobs) == 0 {
		return nil, nil // No high-priority jobs waiting
	}

	// Get the highest priority job (first in the list since they're ordered by priority DESC)
	highPriorityJob := highPriorityJobs[0]

	// Check how many agents are already assigned to this high-priority job
	currentTasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByJobExecution(ctx, highPriorityJob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current tasks for high-priority job: %w", err)
	}
	
	// Count active agents for this job
	activeAgentCount := 0
	for _, task := range currentTasks {
		if task.AgentID != nil && (task.Status == models.JobTaskStatusRunning || task.Status == models.JobTaskStatusAssigned) {
			activeAgentCount++
		}
	}
	
	// Use the job's own max_agents setting (0 means unlimited)
	maxAgents := highPriorityJob.MaxAgents
	if maxAgents == 0 {
		maxAgents = 999 // Treat 0 as unlimited
	}
	
	// Don't interrupt if already at max agents
	if activeAgentCount >= maxAgents {
		debug.Log("High-priority job already at max agents, skipping interruption", map[string]interface{}{
			"job_id": highPriorityJob.ID,
			"active_agents": activeAgentCount,
			"max_agents": maxAgents,
		})
		return nil, nil
	}

	// Check if there are any interruptible jobs with lower priority
	interruptibleJobs, err := s.jobExecutionService.CanInterruptJob(ctx, highPriorityJob.Priority)
	if err != nil {
		return nil, fmt.Errorf("failed to check interruptible jobs: %w", err)
	}

	if len(interruptibleJobs) == 0 {
		return nil, nil // No jobs to interrupt
	}

	// Interrupt the lowest priority job (first in the list since they're ordered by priority ASC)
	lowestPriorityJob := interruptibleJobs[0]
	
	debug.Log("Interrupting lower priority job for high-priority override", map[string]interface{}{
		"interrupted_job_id": lowestPriorityJob.ID,
		"interrupted_priority": lowestPriorityJob.Priority,
		"high_priority_job_id": highPriorityJob.ID,
		"high_priority": highPriorityJob.Priority,
	})

	// Get running tasks for the job to be interrupted
	runningTasks, err := s.jobExecutionService.jobTaskRepo.GetTasksByJobExecution(ctx, lowestPriorityJob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get running tasks: %w", err)
	}

	// Send stop commands to agents for each running task
	for _, task := range runningTasks {
		// Only send stop command if the task has an assigned agent and is running
		if task.AgentID != nil && task.Status == models.JobTaskStatusRunning {
			debug.Log("Sending stop command to agent for task", map[string]interface{}{
				"task_id": task.ID,
				"agent_id": *task.AgentID,
				"job_id": lowestPriorityJob.ID,
			})
			
			// Send stop command via WebSocket integration
			if s.wsIntegration != nil {
				stopErr := s.wsIntegration.SendJobStop(ctx, task.ID, fmt.Sprintf("Job interrupted by higher priority job %s", highPriorityJob.ID))
				if stopErr != nil {
					// Log the error but continue with interruption
					debug.Error("Failed to send stop command to agent %d for task %s: %v", *task.AgentID, task.ID, stopErr)
				}
			}
		}
	}

	// Now interrupt the job in the database
	err = s.jobExecutionService.InterruptJob(ctx, lowestPriorityJob.ID, highPriorityJob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to interrupt job: %w", err)
	}

	return &lowestPriorityJob.ID, nil
}

// ProcessJobCompletion handles job completion and cleanup
func (s *JobSchedulingService) ProcessJobCompletion(ctx context.Context, jobExecutionID uuid.UUID) error {
	debug.Log("Processing job completion", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	// Get job details
	job, err := s.jobExecutionService.jobExecRepo.GetByID(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get job execution: %w", err)
	}

	// Check if all tasks for this job are completed
	incompleteTasks, err := s.jobExecutionService.jobTaskRepo.GetIncompleteTasksCount(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get incomplete tasks count: %w", err)
	}

	// For rule-split jobs, also check if all rules have been dispatched
	if job.UsesRuleSplitting && incompleteTasks == 0 {
		// Get total rules from the job's effective keyspace and base keyspace
		totalRules := job.MultiplicationFactor
		if totalRules == 0 && job.EffectiveKeyspace != nil && job.BaseKeyspace != nil && *job.BaseKeyspace > 0 {
			totalRules = int(*job.EffectiveKeyspace / *job.BaseKeyspace)
		}
		
		// Get the maximum rule end index from all tasks
		maxRuleEnd, err := s.jobExecutionService.jobTaskRepo.GetMaxRuleEndIndex(ctx, jobExecutionID)
		if err != nil {
			debug.Log("Failed to get max rule end index", map[string]interface{}{
				"job_execution_id": jobExecutionID,
				"error":            err.Error(),
			})
		}
		
		// Check if all rules have been processed
		allRulesProcessed := false
		if maxRuleEnd != nil && totalRules > 0 {
			allRulesProcessed = *maxRuleEnd >= totalRules
		}
		
		debug.Log("Rule-split job completion check", map[string]interface{}{
			"job_execution_id":   jobExecutionID,
			"incomplete_tasks":   incompleteTasks,
			"total_rules":        totalRules,
			"max_rule_end":       maxRuleEnd,
			"all_rules_processed": allRulesProcessed,
		})
		
		// Only complete if all tasks are done AND all rules have been dispatched
		if !allRulesProcessed {
			debug.Log("Rule-split job has completed tasks but not all rules dispatched", map[string]interface{}{
				"job_execution_id": jobExecutionID,
				"total_rules":      totalRules,
				"max_rule_end":     maxRuleEnd,
			})
			return nil
		}
	}

	if incompleteTasks == 0 {
		// For non-rule-splitting jobs, verify all effective keyspace has been dispatched
		if !job.UsesRuleSplitting && job.EffectiveKeyspace != nil {
			// Check if all effective keyspace has been dispatched
			if job.DispatchedKeyspace < *job.EffectiveKeyspace {
				debug.Log("Job not complete - more effective keyspace to dispatch", map[string]interface{}{
					"job_id": jobExecutionID,
					"effective_keyspace": *job.EffectiveKeyspace,
					"dispatched_keyspace": job.DispatchedKeyspace,
					"remaining": *job.EffectiveKeyspace - job.DispatchedKeyspace,
					"percentage": float64(job.DispatchedKeyspace) / float64(*job.EffectiveKeyspace) * 100,
				})
				return nil // Don't complete yet, more work to dispatch
			}
		}

		// Also check for regular jobs without multiplication
		if job.MultiplicationFactor <= 1 && job.TotalKeyspace != nil {
			// For jobs without rules, check against total keyspace
			if job.DispatchedKeyspace < *job.TotalKeyspace {
				debug.Log("Job not complete - more keyspace to dispatch", map[string]interface{}{
					"job_id": jobExecutionID,
					"total_keyspace": *job.TotalKeyspace,
					"dispatched_keyspace": job.DispatchedKeyspace,
					"remaining": *job.TotalKeyspace - job.DispatchedKeyspace,
					"percentage": float64(job.DispatchedKeyspace) / float64(*job.TotalKeyspace) * 100,
				})
				return nil // Don't complete yet, more work to dispatch
			}
		}

		// All tasks are complete AND all keyspace has been dispatched, mark job as completed
		err = s.jobExecutionService.CompleteJobExecution(ctx, jobExecutionID)
		if err != nil {
			return fmt.Errorf("failed to complete job execution: %w", err)
		}

		debug.Log("Job execution completed - all tasks done and keyspace fully dispatched", map[string]interface{}{
			"job_execution_id": jobExecutionID,
			"effective_keyspace": job.EffectiveKeyspace,
			"dispatched_keyspace": job.DispatchedKeyspace,
			"incomplete_tasks": incompleteTasks,
		})
	} else {
		debug.Log("Job has incomplete tasks", map[string]interface{}{
			"job_execution_id": jobExecutionID,
			"incomplete_tasks": incompleteTasks,
		})
	}

	return nil
}

// ProcessTaskProgress handles task progress updates and job aggregation
func (s *JobSchedulingService) ProcessTaskProgress(ctx context.Context, taskID uuid.UUID, progress *models.JobProgress) error {
	// Use the enhanced progress tracking method from job execution service
	err := s.jobExecutionService.UpdateTaskProgress(ctx, taskID, progress.KeyspaceProcessed, progress.EffectiveProgress, &progress.HashRate, progress.ProgressPercent)
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

// CleanupStaleAgentStatus cleans up stale agent busy_status metadata
// This runs periodically to catch agents stuck in busy state when database updates failed
func (s *JobSchedulingService) CleanupStaleAgentStatus(ctx context.Context) error {
	debug.Log("Starting stale agent status cleanup", nil)

	// Get all agents with busy_status = "true"
	agents, err := s.agentRepo.List(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}

	cleanedCount := 0
	for _, agent := range agents {
		// Skip if not marked as busy
		if agent.Metadata == nil || agent.Metadata["busy_status"] != "true" {
			continue
		}

		// Check if the agent has a valid running task
		taskIDStr, hasTaskID := agent.Metadata["current_task_id"]
		if !hasTaskID || taskIDStr == "" {
			// No task ID but marked as busy - clear it
			debug.Log("Cleanup: Clearing busy status with no task ID", map[string]interface{}{
				"agent_id": agent.ID,
			})
			agent.Metadata["busy_status"] = "false"
			delete(agent.Metadata, "current_task_id")
			delete(agent.Metadata, "current_job_id")
			if err := s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata); err != nil {
				debug.Error("Failed to clear stale agent status: %v", err)
				continue
			}
			cleanedCount++
			continue
		}

		// Parse and validate task ID
		taskUUID, err := uuid.Parse(taskIDStr)
		if err != nil {
			// Invalid task ID - clear it
			debug.Log("Cleanup: Clearing busy status with invalid task ID", map[string]interface{}{
				"agent_id":     agent.ID,
				"invalid_task": taskIDStr,
			})
			agent.Metadata["busy_status"] = "false"
			delete(agent.Metadata, "current_task_id")
			delete(agent.Metadata, "current_job_id")
			if err := s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata); err != nil {
				debug.Error("Failed to clear stale agent status: %v", err)
				continue
			}
			cleanedCount++
			continue
		}

		// Check if task exists and is actually running
		task, err := s.jobExecutionService.jobTaskRepo.GetByID(ctx, taskUUID)
		if err != nil || task == nil {
			// Task doesn't exist - clear busy status
			debug.Log("Cleanup: Clearing busy status for non-existent task", map[string]interface{}{
				"agent_id": agent.ID,
				"task_id":  taskIDStr,
			})
			agent.Metadata["busy_status"] = "false"
			delete(agent.Metadata, "current_task_id")
			delete(agent.Metadata, "current_job_id")
			if err := s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata); err != nil {
				debug.Error("Failed to clear stale agent status: %v", err)
				continue
			}
			cleanedCount++
			continue
		}

		// Check if task is assigned to this agent and in running state
		if task.AgentID == nil || *task.AgentID != agent.ID {
			debug.Log("Cleanup: Clearing busy status for task assigned to different agent", map[string]interface{}{
				"agent_id":         agent.ID,
				"task_id":          taskIDStr,
				"task_assigned_to": task.AgentID,
			})
			agent.Metadata["busy_status"] = "false"
			delete(agent.Metadata, "current_task_id")
			delete(agent.Metadata, "current_job_id")
			if err := s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata); err != nil {
				debug.Error("Failed to clear stale agent status: %v", err)
				continue
			}
			cleanedCount++
			continue
		}

		if task.Status != models.JobTaskStatusRunning && task.Status != models.JobTaskStatusAssigned {
			debug.Log("Cleanup: Clearing busy status for completed/failed task", map[string]interface{}{
				"agent_id":    agent.ID,
				"task_id":     taskIDStr,
				"task_status": task.Status,
			})
			agent.Metadata["busy_status"] = "false"
			delete(agent.Metadata, "current_task_id")
			delete(agent.Metadata, "current_job_id")
			if err := s.agentRepo.UpdateMetadata(ctx, agent.ID, agent.Metadata); err != nil {
				debug.Error("Failed to clear stale agent status: %v", err)
				continue
			}
			cleanedCount++
			continue
		}

		// If we get here, the agent is legitimately busy
		debug.Log("Agent is legitimately busy", map[string]interface{}{
			"agent_id":    agent.ID,
			"task_id":     taskIDStr,
			"task_status": task.Status,
		})
	}

	if cleanedCount > 0 {
		debug.Log("Stale agent status cleanup completed", map[string]interface{}{
			"agents_cleaned": cleanedCount,
		})
	}

	return nil
}
