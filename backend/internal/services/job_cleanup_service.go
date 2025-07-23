package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// JobCleanupService handles cleanup of stale jobs and tasks
type JobCleanupService struct {
	jobExecutionRepo   *repository.JobExecutionRepository
	jobTaskRepo        *repository.JobTaskRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	agentRepo          *repository.AgentRepository
}

// NewJobCleanupService creates a new job cleanup service
func NewJobCleanupService(
	jobExecutionRepo *repository.JobExecutionRepository,
	jobTaskRepo *repository.JobTaskRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	agentRepo *repository.AgentRepository,
) *JobCleanupService {
	return &JobCleanupService{
		jobExecutionRepo:   jobExecutionRepo,
		jobTaskRepo:        jobTaskRepo,
		systemSettingsRepo: systemSettingsRepo,
		agentRepo:          agentRepo,
	}
}

// CleanupStaleTasksOnStartup cleans up tasks that were left in an incomplete state
func (s *JobCleanupService) CleanupStaleTasksOnStartup(ctx context.Context) error {
	debug.Info("Starting cleanup of stale tasks on startup")

	// Get all tasks that are in assigned or running state
	staleTasks, err := s.jobTaskRepo.GetStaleTasks(ctx)
	if err != nil {
		debug.Error("Failed to get stale tasks: %v", err)
		return fmt.Errorf("failed to get stale tasks: %w", err)
	}

	if len(staleTasks) == 0 {
		debug.Info("No stale tasks found during startup cleanup")
		return nil
	}

	debug.Info("Found stale tasks to cleanup: %d tasks", len(staleTasks))
	for _, task := range staleTasks {
		agentID := 0
		if task.AgentID != nil {
			agentID = *task.AgentID
		}
		debug.Info("Stale task found - ID: %s, Status: %s, Agent: %d, Job: %s",
			task.ID, task.Status, agentID, task.JobExecutionID)
	}

	// Mark each stale task as failed
	for _, task := range staleTasks {
		errorMsg := "Task was incomplete when server restarted - marked as failed"
		err := s.jobTaskRepo.UpdateTaskError(ctx, task.ID, errorMsg)
		if err != nil {
			debug.Error("Failed to update stale task %s: %v", task.ID, err)
			continue
		}

		agentID := 0
		if task.AgentID != nil {
			agentID = *task.AgentID
		}
		debug.Info("Successfully marked stale task as failed - Task ID: %s, Agent: %d, Job: %s, Previous Status: %s",
			task.ID, agentID, task.JobExecutionID, task.Status)
	}

	// Also mark any jobs that were running as pending so they can be rescheduled
	debug.Info("Checking for running jobs to mark as pending...")
	runningJobs, err := s.jobExecutionRepo.GetJobsByStatus(ctx, models.JobExecutionStatusRunning)
	if err != nil {
		debug.Error("Failed to get running jobs: %v", err)
		return fmt.Errorf("failed to get running jobs: %w", err)
	}

	if len(runningJobs) > 0 {
		debug.Info("Found %d running jobs to mark as pending", len(runningJobs))
	} else {
		debug.Info("No running jobs found to mark as pending")
	}

	for _, job := range runningJobs {
		err := s.jobExecutionRepo.UpdateStatus(ctx, job.ID, models.JobExecutionStatusPending)
		if err != nil {
			debug.Error("Failed to mark job %s as pending: %v", job.ID, err)
			continue
		}

		debug.Info("Successfully marked running job as pending for rescheduling - Job ID: %s", job.ID)
	}

	return nil
}

// MonitorStaleTasksPeriodically checks for stale tasks periodically
func (s *JobCleanupService) MonitorStaleTasksPeriodically(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	debug.Log("Starting periodic stale task monitor", map[string]interface{}{
		"interval": interval,
	})

	for {
		select {
		case <-ctx.Done():
			debug.Log("Stale task monitor stopped", nil)
			return
		case <-ticker.C:
			s.checkForStaleTasks(ctx)
		}
	}
}

// checkForStaleTasks checks for tasks that have been assigned/running too long without updates
func (s *JobCleanupService) checkForStaleTasks(ctx context.Context) {
	// Get task timeout setting (default 5 minutes for agent heartbeat)
	taskTimeout := 5 * time.Minute
	timeoutSetting, err := s.systemSettingsRepo.GetSetting(ctx, "task_heartbeat_timeout_minutes")
	if err == nil && timeoutSetting.Value != nil {
		if minutes, err := time.ParseDuration(*timeoutSetting.Value + "m"); err == nil {
			taskTimeout = minutes
		}
	} else {
		// Fall back to task_timeout_minutes if heartbeat setting doesn't exist
		timeoutSetting, err = s.systemSettingsRepo.GetSetting(ctx, "task_timeout_minutes")
		if err == nil && timeoutSetting.Value != nil {
			if minutes, err := time.ParseDuration(*timeoutSetting.Value + "m"); err == nil {
				taskTimeout = minutes
			}
		}
	}

	// Find tasks that haven't been updated in the timeout period
	cutoffTime := time.Now().Add(-taskTimeout)

	staleTasks, err := s.jobTaskRepo.GetTasksNotUpdatedSince(ctx, cutoffTime)
	if err != nil {
		debug.Log("Failed to check for stale tasks", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(staleTasks) == 0 {
		return
	}

	debug.Log("Found stale tasks during periodic check", map[string]interface{}{
		"count":   len(staleTasks),
		"timeout": taskTimeout,
	})

	for _, task := range staleTasks {
		// Check if task has exceeded retry limit (3 attempts)
		if task.RetryCount >= 3 {
			// Mark task as permanently failed
			errorMsg := fmt.Sprintf("Task failed after %d retry attempts (last timeout after %v without progress update)", task.RetryCount, taskTimeout)
			err := s.jobTaskRepo.UpdateTaskError(ctx, task.ID, errorMsg)
			if err != nil {
				debug.Log("Failed to mark stale task as failed", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
				continue
			}

			debug.Log("Marked task as permanently failed after retries", map[string]interface{}{
				"task_id":        task.ID,
				"agent_id":       task.AgentID,
				"retry_count":    task.RetryCount,
				"timeout_period": taskTimeout,
			})

			// Update job's consecutive failures count
			s.updateJobConsecutiveFailures(ctx, task.JobExecutionID, true)

			// Update agent's consecutive failures if assigned
			if task.AgentID != nil {
				s.updateAgentConsecutiveFailures(ctx, *task.AgentID, true)
			}
		} else {
			// Reset task for retry
			err := s.jobTaskRepo.ResetTaskForRetry(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to reset stale task for retry", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
				continue
			}

			debug.Log("Reset timed-out task for retry", map[string]interface{}{
				"task_id":        task.ID,
				"agent_id":       task.AgentID,
				"retry_count":    task.RetryCount + 1,
				"timeout_period": taskTimeout,
			})
		}
	}

	// Check if any affected jobs should be transitioned to pending
	affectedJobs := make(map[uuid.UUID]bool)
	for _, task := range staleTasks {
		affectedJobs[task.JobExecutionID] = true
	}

	for jobID := range affectedJobs {
		// Check if this job has any running or assigned tasks
		allTasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
		if err != nil {
			debug.Log("Failed to check tasks for job", map[string]interface{}{
				"job_id": jobID,
				"error":  err.Error(),
			})
			continue
		}

		// Count active tasks (running or assigned)
		activeTaskCount := 0
		for _, task := range allTasks {
			if task.Status == models.JobTaskStatusRunning || task.Status == models.JobTaskStatusAssigned {
				activeTaskCount++
			}
		}

		// If no active tasks, transition job to pending
		if activeTaskCount == 0 {
			job, err := s.jobExecutionRepo.GetByID(ctx, jobID)
			if err != nil {
				continue
			}

			if job.Status == models.JobExecutionStatusRunning {
				err = s.jobExecutionRepo.UpdateStatus(ctx, jobID, models.JobExecutionStatusPending)
				if err != nil {
					debug.Log("Failed to update job status to pending", map[string]interface{}{
						"job_id": jobID,
						"error":  err.Error(),
					})
					continue
				}

				debug.Log("Updated job status to pending after all tasks timed out", map[string]interface{}{
					"job_id": jobID,
				})
			}
		}
	}
}

// updateJobConsecutiveFailures updates the consecutive failure count for a job
func (s *JobCleanupService) updateJobConsecutiveFailures(ctx context.Context, jobExecutionID uuid.UUID, failed bool) {
	jobExecution, err := s.jobExecutionRepo.GetByID(ctx, jobExecutionID)
	if err != nil {
		debug.Log("Failed to get job execution for failure tracking", map[string]interface{}{
			"job_execution_id": jobExecutionID,
			"error":            err.Error(),
		})
		return
	}

	if failed {
		// Increment consecutive failures
		newCount := jobExecution.ConsecutiveFailures + 1
		err = s.jobExecutionRepo.UpdateConsecutiveFailures(ctx, jobExecutionID, newCount)
		if err != nil {
			debug.Log("Failed to update job consecutive failures", map[string]interface{}{
				"job_execution_id": jobExecutionID,
				"error":            err.Error(),
			})
			return
		}

		// Check if we've hit the threshold (3 consecutive failures)
		if newCount >= 3 {
			// Mark the entire job as failed
			err = s.jobExecutionRepo.UpdateStatus(ctx, jobExecutionID, models.JobExecutionStatusFailed)
			if err != nil {
				debug.Log("Failed to mark job as failed", map[string]interface{}{
					"job_execution_id": jobExecutionID,
					"error":            err.Error(),
				})
				return
			}

			errorMsg := fmt.Sprintf("Job failed due to %d consecutive task failures", newCount)
			err = s.jobExecutionRepo.UpdateErrorMessage(ctx, jobExecutionID, errorMsg)
			if err != nil {
				debug.Log("Failed to update job error message", map[string]interface{}{
					"job_execution_id": jobExecutionID,
					"error":            err.Error(),
				})
			}

			debug.Log("Marked job as failed due to consecutive failures", map[string]interface{}{
				"job_execution_id":     jobExecutionID,
				"consecutive_failures": newCount,
			})
		}
	} else {
		// Reset consecutive failures on success
		if jobExecution.ConsecutiveFailures > 0 {
			err = s.jobExecutionRepo.UpdateConsecutiveFailures(ctx, jobExecutionID, 0)
			if err != nil {
				debug.Log("Failed to reset job consecutive failures", map[string]interface{}{
					"job_execution_id": jobExecutionID,
					"error":            err.Error(),
				})
			}
		}
	}
}

// updateAgentConsecutiveFailures updates the consecutive failure count for an agent
func (s *JobCleanupService) updateAgentConsecutiveFailures(ctx context.Context, agentID int, failed bool) {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		debug.Log("Failed to get agent for failure tracking", map[string]interface{}{
			"agent_id": agentID,
			"error":    err.Error(),
		})
		return
	}

	if failed {
		// Increment consecutive failures
		newCount := agent.ConsecutiveFailures + 1
		err = s.agentRepo.UpdateConsecutiveFailures(ctx, agentID, newCount)
		if err != nil {
			debug.Log("Failed to update agent consecutive failures", map[string]interface{}{
				"agent_id": agentID,
				"error":    err.Error(),
			})
			return
		}

		// Check if we've hit the threshold (3 consecutive failures)
		if newCount >= 3 {
			// Mark the agent as unhealthy/error state
			errorMsg := fmt.Sprintf("Agent has %d consecutive task failures", newCount)
			err = s.agentRepo.UpdateStatus(ctx, agentID, models.AgentStatusError, &errorMsg)
			if err != nil {
				debug.Log("Failed to mark agent as error state", map[string]interface{}{
					"agent_id": agentID,
					"error":    err.Error(),
				})
				return
			}

			debug.Log("Marked agent as error state due to consecutive failures", map[string]interface{}{
				"agent_id":             agentID,
				"consecutive_failures": newCount,
			})
		}
	} else {
		// Reset consecutive failures on success
		if agent.ConsecutiveFailures > 0 {
			err = s.agentRepo.UpdateConsecutiveFailures(ctx, agentID, 0)
			if err != nil {
				debug.Log("Failed to reset agent consecutive failures", map[string]interface{}{
					"agent_id": agentID,
					"error":    err.Error(),
				})
			}
		}
	}
}
