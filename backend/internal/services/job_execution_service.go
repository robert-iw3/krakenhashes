package services

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)


// JobExecutionService handles job execution orchestration
type JobExecutionService struct {
	jobExecRepo       *repository.JobExecutionRepository
	jobTaskRepo       *repository.JobTaskRepository
	benchmarkRepo     *repository.BenchmarkRepository
	agentHashlistRepo *repository.AgentHashlistRepository
	agentRepo         *repository.AgentRepository
	presetJobRepo     repository.PresetJobRepository
	hashlistRepo      *repository.HashListRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	
	// Configuration paths
	hashcatBinaryPath string
	dataDirectory     string
}

// NewJobExecutionService creates a new job execution service
func NewJobExecutionService(
	jobExecRepo *repository.JobExecutionRepository,
	jobTaskRepo *repository.JobTaskRepository,
	benchmarkRepo *repository.BenchmarkRepository,
	agentHashlistRepo *repository.AgentHashlistRepository,
	agentRepo *repository.AgentRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	hashcatBinaryPath string,
	dataDirectory string,
) *JobExecutionService {
	return &JobExecutionService{
		jobExecRepo:        jobExecRepo,
		jobTaskRepo:        jobTaskRepo,
		benchmarkRepo:      benchmarkRepo,
		agentHashlistRepo:  agentHashlistRepo,
		agentRepo:          agentRepo,
		presetJobRepo:      presetJobRepo,
		hashlistRepo:       hashlistRepo,
		systemSettingsRepo: systemSettingsRepo,
		hashcatBinaryPath:  hashcatBinaryPath,
		dataDirectory:      dataDirectory,
	}
}

// CreateJobExecution creates a new job execution from a preset job and hashlist
func (s *JobExecutionService) CreateJobExecution(ctx context.Context, presetJobID uuid.UUID, hashlistID int64) (*models.JobExecution, error) {
	debug.Log("Creating job execution", map[string]interface{}{
		"preset_job_id": presetJobID,
		"hashlist_id":   hashlistID,
	})

	// Get the preset job
	presetJob, err := s.presetJobRepo.GetByID(ctx, presetJobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preset job: %w", err)
	}

	// Get the hashlist
	hashlist, err := s.hashlistRepo.GetByID(ctx, hashlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Calculate total keyspace
	totalKeyspace, err := s.calculateKeyspace(ctx, presetJob, hashlist)
	if err != nil {
		debug.Log("Failed to calculate keyspace", map[string]interface{}{
			"preset_job_id": presetJobID,
			"hashlist_id":   hashlistID,
			"error":         err.Error(),
		})
		// Continue without keyspace - some attacks don't support it
	}

	// Create job execution
	jobExecution := &models.JobExecution{
		PresetJobID:       presetJobID,
		HashlistID:        hashlistID,
		Status:            models.JobExecutionStatusPending,
		Priority:          presetJob.Priority,
		TotalKeyspace:     totalKeyspace,
		ProcessedKeyspace: 0,
		AttackMode:        presetJob.AttackMode,
	}

	err = s.jobExecRepo.Create(ctx, jobExecution)
	if err != nil {
		return nil, fmt.Errorf("failed to create job execution: %w", err)
	}

	debug.Log("Job execution created", map[string]interface{}{
		"job_execution_id": jobExecution.ID,
		"total_keyspace":   totalKeyspace,
	})

	return jobExecution, nil
}

// calculateKeyspace calculates the total keyspace for a job using hashcat --keyspace
func (s *JobExecutionService) calculateKeyspace(ctx context.Context, presetJob *models.PresetJob, hashlist *models.HashList) (*int64, error) {
	// Build hashcat command for keyspace calculation
	args := []string{
		"-m", strconv.Itoa(hashlist.HashTypeID),
		"-a", strconv.Itoa(int(presetJob.AttackMode)),
	}

	// Add dummy hash file (hashcat needs it for keyspace calculation)
	hashFilePath := filepath.Join(s.dataDirectory, "hashlists", fmt.Sprintf("%d.hash", hashlist.ID))
	args = append(args, hashFilePath)

	// Add attack-specific arguments
	switch presetJob.AttackMode {
	case models.AttackModeStraight: // Dictionary attack
		// Add wordlists
		for _, wordlistIDStr := range presetJob.WordlistIDs {
			wordlistPath := filepath.Join(s.dataDirectory, "wordlists", "custom", fmt.Sprintf("%s.txt", wordlistIDStr))
			args = append(args, wordlistPath)
		}
		// Add rules if any
		for _, ruleIDStr := range presetJob.RuleIDs {
			rulePath := filepath.Join(s.dataDirectory, "rules", "custom", fmt.Sprintf("%s.rule", ruleIDStr))
			args = append(args, "-r", rulePath)
		}

	case models.AttackModeCombination: // Combinator attack
		if len(presetJob.WordlistIDs) >= 2 {
			wordlist1Path := filepath.Join(s.dataDirectory, "wordlists", "custom", fmt.Sprintf("%s.txt", presetJob.WordlistIDs[0]))
			wordlist2Path := filepath.Join(s.dataDirectory, "wordlists", "custom", fmt.Sprintf("%s.txt", presetJob.WordlistIDs[1]))
			args = append(args, wordlist1Path, wordlist2Path)
		}

	case models.AttackModeBruteForce: // Mask attack
		if presetJob.Mask != "" {
			args = append(args, presetJob.Mask)
		}

	case models.AttackModeHybridWordlistMask: // Hybrid Wordlist + Mask
		if len(presetJob.WordlistIDs) > 0 && presetJob.Mask != "" {
			wordlistPath := filepath.Join(s.dataDirectory, "wordlists", "custom", fmt.Sprintf("%s.txt", presetJob.WordlistIDs[0]))
			args = append(args, wordlistPath, presetJob.Mask)
		}

	case models.AttackModeHybridMaskWordlist: // Hybrid Mask + Wordlist
		if presetJob.Mask != "" && len(presetJob.WordlistIDs) > 0 {
			wordlistPath := filepath.Join(s.dataDirectory, "wordlists", "custom", fmt.Sprintf("%s.txt", presetJob.WordlistIDs[0]))
			args = append(args, presetJob.Mask, wordlistPath)
		}

	default:
		return nil, fmt.Errorf("unsupported attack mode for keyspace calculation: %d", presetJob.AttackMode)
	}

	// Add keyspace flag
	args = append(args, "--keyspace")

	debug.Log("Calculating keyspace", map[string]interface{}{
		"command": s.hashcatBinaryPath,
		"args":    args,
	})

	// Execute hashcat command with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.hashcatBinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("hashcat keyspace calculation failed: %w", err)
	}

	// Parse keyspace from output
	keyspaceStr := strings.TrimSpace(string(output))
	keyspace, err := strconv.ParseInt(keyspaceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyspace: %w", err)
	}

	if keyspace <= 0 {
		return nil, fmt.Errorf("invalid keyspace: %d", keyspace)
	}

	return &keyspace, nil
}

// GetNextPendingJob returns the next job to be executed based on priority and FIFO
func (s *JobExecutionService) GetNextPendingJob(ctx context.Context) (*models.JobExecution, error) {
	pendingJobs, err := s.jobExecRepo.GetPendingJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	if len(pendingJobs) == 0 {
		return nil, nil // No pending jobs
	}

	// Jobs are already ordered by priority DESC, created_at ASC in the repository
	return &pendingJobs[0], nil
}

// GetAvailableAgents returns agents that are available to take on new work
func (s *JobExecutionService) GetAvailableAgents(ctx context.Context) ([]models.Agent, error) {
	// Get max concurrent jobs per agent setting
	maxConcurrentSetting, err := s.systemSettingsRepo.GetSetting(ctx, "max_concurrent_jobs_per_agent")
	if err != nil {
		return nil, fmt.Errorf("failed to get max concurrent jobs setting: %w", err)
	}

	maxConcurrent := 2 // Default value
	if maxConcurrentSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*maxConcurrentSetting.Value); parseErr == nil {
			maxConcurrent = parsed
		}
	}

	// Get all active agents
	agents, err := s.agentRepo.List(ctx, map[string]interface{}{"status": models.AgentStatusActive})
	if err != nil {
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	var availableAgents []models.Agent
	for _, agent := range agents {
		// Count active tasks for this agent
		activeTasks, err := s.jobTaskRepo.GetActiveTasksByAgent(ctx, agent.ID)
		if err != nil {
			debug.Log("Failed to get active tasks for agent", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    err.Error(),
			})
			continue
		}

		if len(activeTasks) < maxConcurrent {
			availableAgents = append(availableAgents, agent)
		}
	}

	return availableAgents, nil
}

// CreateJobTask creates a task chunk for an agent
func (s *JobExecutionService) CreateJobTask(ctx context.Context, jobExecution *models.JobExecution, agent *models.Agent, keyspaceStart, keyspaceEnd int64, benchmarkSpeed *int64) (*models.JobTask, error) {
	// Get chunk duration setting
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk duration setting: %w", err)
	}

	chunkDuration := 1200 // Default 20 minutes
	if chunkDurationSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*chunkDurationSetting.Value); parseErr == nil {
			chunkDuration = parsed
		}
	}

	jobTask := &models.JobTask{
		JobExecutionID:    jobExecution.ID,
		AgentID:           agent.ID,
		Status:            models.JobTaskStatusPending,
		KeyspaceStart:     keyspaceStart,
		KeyspaceEnd:       keyspaceEnd,
		KeyspaceProcessed: 0,
		BenchmarkSpeed:    benchmarkSpeed,
		ChunkDuration:     chunkDuration,
	}

	err = s.jobTaskRepo.Create(ctx, jobTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create job task: %w", err)
	}

	debug.Log("Job task created", map[string]interface{}{
		"task_id":         jobTask.ID,
		"agent_id":        agent.ID,
		"keyspace_start":  keyspaceStart,
		"keyspace_end":    keyspaceEnd,
		"chunk_duration":  chunkDuration,
	})

	return jobTask, nil
}

// StartJobExecution marks a job execution as started
func (s *JobExecutionService) StartJobExecution(ctx context.Context, jobExecutionID uuid.UUID) error {
	err := s.jobExecRepo.StartExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to start job execution: %w", err)
	}

	debug.Log("Job execution started", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	return nil
}

// CompleteJobExecution marks a job execution as completed
func (s *JobExecutionService) CompleteJobExecution(ctx context.Context, jobExecutionID uuid.UUID) error {
	err := s.jobExecRepo.CompleteExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to complete job execution: %w", err)
	}

	debug.Log("Job execution completed", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	return nil
}

// UpdateJobProgress updates the progress of a job execution
func (s *JobExecutionService) UpdateJobProgress(ctx context.Context, jobExecutionID uuid.UUID, processedKeyspace int64) error {
	err := s.jobExecRepo.UpdateProgress(ctx, jobExecutionID, processedKeyspace)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// CanInterruptJob checks if a job can be interrupted by a higher priority job
func (s *JobExecutionService) CanInterruptJob(ctx context.Context, newJobPriority int) ([]models.JobExecution, error) {
	// Check if job interruption is enabled
	interruptionSetting, err := s.systemSettingsRepo.GetSetting(ctx, "job_interruption_enabled")
	if err != nil {
		return nil, fmt.Errorf("failed to get interruption setting: %w", err)
	}

	if interruptionSetting.Value == nil || *interruptionSetting.Value != "true" {
		return []models.JobExecution{}, nil // Interruption disabled
	}

	// Get interruptible jobs with lower priority
	interruptibleJobs, err := s.jobExecRepo.GetInterruptibleJobs(ctx, newJobPriority)
	if err != nil {
		return nil, fmt.Errorf("failed to get interruptible jobs: %w", err)
	}

	return interruptibleJobs, nil
}

// InterruptJob interrupts a running job for a higher priority job
func (s *JobExecutionService) InterruptJob(ctx context.Context, jobExecutionID, interruptingJobID uuid.UUID) error {
	err := s.jobExecRepo.InterruptExecution(ctx, jobExecutionID, interruptingJobID)
	if err != nil {
		return fmt.Errorf("failed to interrupt job: %w", err)
	}

	// Cancel all running tasks for this job
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get tasks for interrupted job: %w", err)
	}

	for _, task := range tasks {
		if task.Status == models.JobTaskStatusRunning {
			err = s.jobTaskRepo.CancelTask(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to cancel task", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
			}
		}
	}

	debug.Log("Job interrupted", map[string]interface{}{
		"job_execution_id":      jobExecutionID,
		"interrupting_job_id":   interruptingJobID,
	})

	return nil
}
// GetSystemSetting retrieves a system setting by key (public method for integration)
func (s *JobExecutionService) GetSystemSetting(ctx context.Context, key string) (int, error) {
	setting, err := s.systemSettingsRepo.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	
	if setting.Value == nil {
		return 0, fmt.Errorf("setting value is null")
	}
	
	value, err := strconv.Atoi(*setting.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid setting value: %w", err)
	}
	
	return value, nil
}

// GetJobExecutionByID retrieves a job execution by ID (public method for integration)
func (s *JobExecutionService) GetJobExecutionByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	return s.jobExecRepo.GetByID(ctx, id)
}
