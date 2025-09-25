package integration

import (
	"context"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// JobSchedulerStarter manages the lifecycle of the job scheduler
type JobSchedulerStarter struct {
	jobIntegration *JobIntegrationManager
	scheduleTicker *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	started        bool
}

// NewJobSchedulerStarter creates a new job scheduler starter
func NewJobSchedulerStarter(jobIntegration *JobIntegrationManager) *JobSchedulerStarter {
	return &JobSchedulerStarter{
		jobIntegration: jobIntegration,
	}
}

// Start begins the job scheduling loop
func (s *JobSchedulerStarter) Start(ctx context.Context) error {
	if s.started {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true

	// Get scheduling interval from settings (default 3 seconds)
	interval := 3 * time.Second
	s.scheduleTicker = time.NewTicker(interval)

	debug.Log("Starting job scheduler with interval", map[string]interface{}{
		"interval": interval.String(),
	})

	// Start scheduler in background
	go s.run()

	return nil
}

// Stop gracefully stops the job scheduler
func (s *JobSchedulerStarter) Stop() {
	if !s.started {
		return
	}

	debug.Log("Stopping job scheduler", nil)

	if s.scheduleTicker != nil {
		s.scheduleTicker.Stop()
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.started = false
}

// run is the main scheduling loop
func (s *JobSchedulerStarter) run() {
	debug.Log("Job scheduler loop started", nil)

	// Run initial scheduling immediately
	s.runSchedulingCycle()

	for {
		select {
		case <-s.ctx.Done():
			debug.Log("Job scheduler context cancelled", nil)
			return
		case <-s.scheduleTicker.C:
			s.runSchedulingCycle()
		}
	}
}

// runSchedulingCycle executes one scheduling cycle
func (s *JobSchedulerStarter) runSchedulingCycle() {
	// Check if we have any connected agents
	connectedAgents := s.jobIntegration.GetConnectedAgentCount()
	if connectedAgents == 0 {
		debug.Log("No connected agents, skipping scheduling cycle", nil)
		return
	}

	debug.Log("Running job scheduling cycle", map[string]interface{}{
		"connected_agents": connectedAgents,
	})

	// Create a timeout context for the scheduling operation
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Run the scheduling logic
	result, err := s.jobIntegration.jobSchedulingService.ScheduleJobs(ctx)
	if err != nil {
		debug.Log("Job scheduling cycle failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	debug.Log("Job scheduling cycle completed", map[string]interface{}{
		"assigned_tasks":   len(result.AssignedTasks),
		"interrupted_jobs": len(result.InterruptedJobs),
		"errors":           len(result.Errors),
	})

	// Log any errors that occurred
	for _, err := range result.Errors {
		debug.Log("Scheduling error", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// GetJobIntegration returns the job integration manager
func (s *JobSchedulerStarter) GetJobIntegration() *JobIntegrationManager {
	return s.jobIntegration
}
