package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// JobCleanupService handles cleanup of stale jobs and tasks
type JobCleanupService struct {
	jobExecutionRepo   *repository.JobExecutionRepository
	jobTaskRepo        *repository.JobTaskRepository
	systemSettingsRepo *repository.SystemSettingsRepository
}

// NewJobCleanupService creates a new job cleanup service
func NewJobCleanupService(
	jobExecutionRepo *repository.JobExecutionRepository,
	jobTaskRepo *repository.JobTaskRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
) *JobCleanupService {
	return &JobCleanupService{
		jobExecutionRepo:   jobExecutionRepo,
		jobTaskRepo:        jobTaskRepo,
		systemSettingsRepo: systemSettingsRepo,
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
		debug.Info("Stale task found - ID: %s, Status: %s, Agent: %d, Job: %s", 
			task.ID, task.Status, task.AgentID, task.JobExecutionID)
	}

	// Mark each stale task as failed
	for _, task := range staleTasks {
		errorMsg := "Task was incomplete when server restarted - marked as failed"
		err := s.jobTaskRepo.UpdateTaskError(ctx, task.ID, errorMsg)
		if err != nil {
			debug.Error("Failed to update stale task %s: %v", task.ID, err)
			continue
		}

		debug.Info("Successfully marked stale task as failed - Task ID: %s, Agent: %d, Job: %s, Previous Status: %s",
			task.ID, task.AgentID, task.JobExecutionID, task.Status)
	}

	// Also mark any jobs that were running as interrupted
	debug.Info("Checking for running jobs to mark as interrupted...")
	runningJobs, err := s.jobExecutionRepo.GetJobsByStatus(ctx, models.JobExecutionStatusRunning)
	if err != nil {
		debug.Error("Failed to get running jobs: %v", err)
		return fmt.Errorf("failed to get running jobs: %w", err)
	}

	if len(runningJobs) > 0 {
		debug.Info("Found %d running jobs to mark as interrupted", len(runningJobs))
	} else {
		debug.Info("No running jobs found to mark as interrupted")
	}

	for _, job := range runningJobs {
		err := s.jobExecutionRepo.UpdateStatus(ctx, job.ID, models.JobExecutionStatusInterrupted)
		if err != nil {
			debug.Error("Failed to mark job %s as interrupted: %v", job.ID, err)
			continue
		}

		debug.Info("Successfully marked running job as interrupted - Job ID: %s", job.ID)
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
	// Get task timeout setting (default 30 minutes)
	taskTimeout := 30 * time.Minute
	timeoutSetting, err := s.systemSettingsRepo.GetSetting(ctx, "task_timeout_minutes")
	if err == nil && timeoutSetting.Value != nil {
		if minutes, err := time.ParseDuration(*timeoutSetting.Value + "m"); err == nil {
			taskTimeout = minutes
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
		// Check if the agent is still connected
		// If not, mark the task as failed
		errorMsg := fmt.Sprintf("Task timed out after %v without progress update", taskTimeout)
		err := s.jobTaskRepo.UpdateTaskError(ctx, task.ID, errorMsg)
		if err != nil {
			debug.Log("Failed to mark stale task as failed", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			continue
		}

		debug.Log("Marked timed-out task as failed", map[string]interface{}{
			"task_id":        task.ID,
			"agent_id":       task.AgentID,
			"last_update":    task.UpdatedAt,
			"timeout_period": taskTimeout,
		})
	}
}

// RecoverInterruptedJobs attempts to reschedule interrupted jobs
func (s *JobCleanupService) RecoverInterruptedJobs(ctx context.Context) error {
	// Get all interrupted jobs
	interruptedJobs, err := s.jobExecutionRepo.GetJobsByStatus(ctx, models.JobExecutionStatusInterrupted)
	if err != nil {
		return fmt.Errorf("failed to get interrupted jobs: %w", err)
	}

	if len(interruptedJobs) == 0 {
		return nil
	}

	debug.Log("Found interrupted jobs to recover", map[string]interface{}{
		"count": len(interruptedJobs),
	})

	// Mark them as pending so they can be rescheduled
	for _, job := range interruptedJobs {
		err := s.jobExecutionRepo.UpdateStatus(ctx, job.ID, models.JobExecutionStatusPending)
		if err != nil {
			debug.Log("Failed to recover interrupted job", map[string]interface{}{
				"job_id": job.ID,
				"error":  err.Error(),
			})
			continue
		}

		debug.Log("Recovered interrupted job", map[string]interface{}{
			"job_id":       job.ID,
			"hashlist_id":  job.HashlistID,
			"preset_job":   job.PresetJobID,
		})
	}

	return nil
}