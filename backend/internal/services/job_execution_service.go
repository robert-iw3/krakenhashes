package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
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
	deviceRepo        *repository.AgentDeviceRepository
	presetJobRepo     repository.PresetJobRepository
	hashlistRepo      *repository.HashListRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	fileRepo          *repository.FileRepository
	binaryManager     binary.Manager
	
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
	deviceRepo *repository.AgentDeviceRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	fileRepo *repository.FileRepository,
	binaryManager binary.Manager,
	hashcatBinaryPath string,
	dataDirectory string,
) *JobExecutionService {
	debug.Log("Creating JobExecutionService", map[string]interface{}{
		"data_directory": dataDirectory,
		"is_absolute":    filepath.IsAbs(dataDirectory),
	})
	
	return &JobExecutionService{
		jobExecRepo:        jobExecRepo,
		jobTaskRepo:        jobTaskRepo,
		benchmarkRepo:      benchmarkRepo,
		agentHashlistRepo:  agentHashlistRepo,
		agentRepo:          agentRepo,
		deviceRepo:         deviceRepo,
		presetJobRepo:      presetJobRepo,
		hashlistRepo:       hashlistRepo,
		systemSettingsRepo: systemSettingsRepo,
		fileRepo:           fileRepo,
		binaryManager:      binaryManager,
		hashcatBinaryPath:  hashcatBinaryPath,
		dataDirectory:      dataDirectory,
	}
}

// CreateJobExecution creates a new job execution from a preset job and hashlist
func (s *JobExecutionService) CreateJobExecution(ctx context.Context, presetJobID uuid.UUID, hashlistID int64, createdBy *uuid.UUID) (*models.JobExecution, error) {
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

	// Use pre-calculated keyspace from preset job if available
	var totalKeyspace *int64
	if presetJob.Keyspace != nil && *presetJob.Keyspace > 0 {
		totalKeyspace = presetJob.Keyspace
		debug.Log("Using pre-calculated keyspace from preset job", map[string]interface{}{
			"preset_job_id": presetJobID,
			"keyspace": *totalKeyspace,
		})
	} else {
		// Fallback to calculating keyspace if not pre-calculated
		debug.Warning("Preset job has no pre-calculated keyspace, calculating now")
		totalKeyspace, err = s.calculateKeyspace(ctx, presetJob, hashlist)
		if err != nil {
			debug.Error("Failed to calculate keyspace: %v", err)
			return nil, fmt.Errorf("keyspace calculation is required for job execution: %w", err)
		}
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
		MaxAgents:         presetJob.MaxAgents,
		CreatedBy:         createdBy,
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
	// Get the hashcat binary path from binary manager
	hashcatPath, err := s.binaryManager.GetLocalBinaryPath(ctx, int64(presetJob.BinaryVersionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get hashcat binary path: %w", err)
	}

	// Build hashcat command for keyspace calculation
	// For keyspace calculation, we don't need -m (hash type) or the hash file
	// We only need the attack-specific inputs
	var args []string

	// Add attack-specific arguments
	switch presetJob.AttackMode {
	case models.AttackModeStraight: // Dictionary attack (-a 0)
		// For straight attack, only need wordlist(s) and optionally rules
		// The keyspace is the number of words in the wordlist (or with rules applied)
		for _, wordlistIDStr := range presetJob.WordlistIDs {
			wordlistPath, err := s.resolveWordlistPath(ctx, wordlistIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath)
		}
		// Add rules if any (rules don't change the keyspace command, but hashcat will calculate accordingly)
		for _, ruleIDStr := range presetJob.RuleIDs {
			rulePath, err := s.resolveRulePath(ctx, ruleIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve rule path: %w", err)
			}
			args = append(args, "-r", rulePath)
		}

	case models.AttackModeCombination: // Combinator attack
		if len(presetJob.WordlistIDs) >= 2 {
			wordlist1Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist1 path: %w", err)
			}
			wordlist2Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[1])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist2 path: %w", err)
			}
			args = append(args, wordlist1Path, wordlist2Path)
		}

	case models.AttackModeBruteForce: // Mask attack
		if presetJob.Mask != "" {
			args = append(args, presetJob.Mask)
		}

	case models.AttackModeHybridWordlistMask: // Hybrid Wordlist + Mask
		if len(presetJob.WordlistIDs) > 0 && presetJob.Mask != "" {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath, presetJob.Mask)
		}

	case models.AttackModeHybridMaskWordlist: // Hybrid Mask + Wordlist
		if presetJob.Mask != "" && len(presetJob.WordlistIDs) > 0 {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, presetJob.Mask, wordlistPath)
		}

	default:
		return nil, fmt.Errorf("unsupported attack mode for keyspace calculation: %d", presetJob.AttackMode)
	}

	// Add keyspace flag
	args = append(args, "--keyspace")

	debug.Log("Calculating keyspace", map[string]interface{}{
		"command": hashcatPath,
		"args":    args,
		"attack_mode": presetJob.AttackMode,
	})

	// Execute hashcat command with timeout
	// Increase timeout to 2 minutes to allow for large wordlist processing
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(ctx, hashcatPath, args...)
	
	// Log current working directory for debugging
	cwd, _ := os.Getwd()
	debug.Log("Executing hashcat command", map[string]interface{}{
		"working_dir": cwd,
		"command":     hashcatPath,
		"args":        args,
	})
	
	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	if err != nil {
		// Log the full output for debugging
		debug.Error("Hashcat keyspace calculation failed", map[string]interface{}{
			"error":       err.Error(),
			"stdout":      stdout.String(),
			"stderr":      stderr.String(),
			"working_dir": cwd,
			"command":     hashcatPath,
			"args":        args,
		})
		return nil, fmt.Errorf("hashcat keyspace calculation failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Parse keyspace from output
	// The keyspace should be the last line of stdout (ignoring stderr warnings about invalid rules)
	outputLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(outputLines) == 0 {
		return nil, fmt.Errorf("no output from hashcat keyspace calculation")
	}
	
	// Get the last non-empty line
	var keyspaceStr string
	for i := len(outputLines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(outputLines[i])
		if line != "" {
			keyspaceStr = line
			break
		}
	}
	
	keyspace, err := strconv.ParseInt(keyspaceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyspace '%s': %w", keyspaceStr, err)
	}

	if keyspace <= 0 {
		return nil, fmt.Errorf("invalid keyspace: %d", keyspace)
	}
	
	duration := time.Since(startTime)
	debug.Log("Keyspace calculated successfully", map[string]interface{}{
		"keyspace": keyspace,
		"duration": duration.String(),
		"stderr_warnings": stderr.String(),
	})

	return &keyspace, nil
}

// GetNextPendingJob returns the next job to be executed based on priority and FIFO
func (s *JobExecutionService) GetNextPendingJob(ctx context.Context) (*models.JobExecution, error) {
	debug.Log("Getting next pending job", nil)
	
	pendingJobs, err := s.jobExecRepo.GetPendingJobs(ctx)
	if err != nil {
		debug.Log("Failed to get pending jobs from repository", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	debug.Log("Retrieved pending jobs", map[string]interface{}{
		"count": len(pendingJobs),
	})

	if len(pendingJobs) == 0 {
		return nil, nil // No pending jobs
	}

	// Jobs are already ordered by priority DESC, created_at ASC in the repository
	nextJob := &pendingJobs[0]
	debug.Log("Selected next job", map[string]interface{}{
		"job_id":        nextJob.ID,
		"priority":      nextJob.Priority,
		"preset_job":    nextJob.PresetJobName,
		"hashlist":      nextJob.HashlistName,
	})
	
	return nextJob, nil
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

	debug.Log("Found active agents", map[string]interface{}{
		"agent_count": len(agents),
	})

	var availableAgents []models.Agent
	for _, agent := range agents {
		debug.Log("Checking agent availability", map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"status":     agent.Status,
			"is_enabled": agent.IsEnabled,
		})
		
		// Skip disabled agents (maintenance mode)
		if !agent.IsEnabled {
			debug.Log("Agent is disabled (maintenance mode), skipping", map[string]interface{}{
				"agent_id": agent.ID,
			})
			continue
		}
		
		// Count active tasks for this agent
		activeTasks, err := s.jobTaskRepo.GetActiveTasksByAgent(ctx, agent.ID)
		if err != nil {
			debug.Log("Failed to get active tasks for agent", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    err.Error(),
			})
			continue
		}

		debug.Log("Agent task status", map[string]interface{}{
			"agent_id":       agent.ID,
			"active_tasks":   len(activeTasks),
			"max_concurrent": maxConcurrent,
			"is_available":   len(activeTasks) < maxConcurrent,
		})

		if len(activeTasks) < maxConcurrent {
			// Check if agent has enabled devices
			hasEnabledDevices, err := s.deviceRepo.HasEnabledDevices(agent.ID)
			if err != nil {
				debug.Log("Failed to check enabled devices for agent", map[string]interface{}{
					"agent_id": agent.ID,
					"error":    err.Error(),
				})
				continue
			}
			
			if hasEnabledDevices {
				availableAgents = append(availableAgents, agent)
			} else {
				debug.Log("Agent has no enabled devices, skipping", map[string]interface{}{
					"agent_id": agent.ID,
				})
			}
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

// UpdateCrackedCount updates the total number of cracked hashes for a job execution
// DEPRECATED: This method is deprecated as cracked counts are now tracked at the hashlist level
func (s *JobExecutionService) UpdateCrackedCount(ctx context.Context, jobExecutionID uuid.UUID, additionalCracks int) error {
	// This method is deprecated and should not be used
	// Cracked counts are now tracked on the hashlists table, not job_executions
	debug.Log("WARNING: UpdateCrackedCount called on job execution service (deprecated)", map[string]interface{}{
		"job_id":            jobExecutionID,
		"additional_cracks": additionalCracks,
	})
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

// RetryFailedChunk attempts to retry a failed job task chunk
func (s *JobExecutionService) RetryFailedChunk(ctx context.Context, taskID uuid.UUID) error {
	debug.Log("Attempting to retry failed chunk", map[string]interface{}{
		"task_id": taskID,
	})

	// Get the current task
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get max retry attempts from system settings
	maxRetryAttempts, err := s.GetSystemSetting(ctx, "max_chunk_retry_attempts")
	if err != nil {
		debug.Log("Failed to get max retry attempts, using default", map[string]interface{}{
			"error": err.Error(),
		})
		maxRetryAttempts = 3 // Default fallback
	}

	// Check if we can retry
	if task.RetryCount >= maxRetryAttempts {
		debug.Log("Maximum retry attempts reached", map[string]interface{}{
			"task_id":     taskID,
			"retry_count": task.RetryCount,
			"max_retries": maxRetryAttempts,
		})
		
		// Mark as permanently failed
		err = s.jobTaskRepo.UpdateTaskStatus(ctx, taskID, "failed", "failed")
		if err != nil {
			return fmt.Errorf("failed to mark task as permanently failed: %w", err)
		}
		
		return fmt.Errorf("maximum retry attempts (%d) exceeded for task %s", maxRetryAttempts, taskID)
	}

	// Reset task for retry
	err = s.jobTaskRepo.ResetTaskForRetry(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to reset task for retry: %w", err)
	}

	debug.Log("Chunk reset for retry", map[string]interface{}{
		"task_id":     taskID,
		"retry_count": task.RetryCount + 1,
	})

	return nil
}

// ProcessFailedChunks automatically retries failed chunks based on system settings
func (s *JobExecutionService) ProcessFailedChunks(ctx context.Context, jobExecutionID uuid.UUID) error {
	debug.Log("Processing failed chunks for job", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	// Get all failed tasks for this job execution
	failedTasks, err := s.jobTaskRepo.GetFailedTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get failed tasks: %w", err)
	}

	retriedCount := 0
	permanentFailureCount := 0

	for _, task := range failedTasks {
		err := s.RetryFailedChunk(ctx, task.ID)
		if err != nil {
			debug.Log("Failed to retry chunk", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			permanentFailureCount++
		} else {
			retriedCount++
		}
	}

	debug.Log("Completed failed chunk processing", map[string]interface{}{
		"job_execution_id":       jobExecutionID,
		"retried_count":          retriedCount,
		"permanent_failure_count": permanentFailureCount,
		"total_failed_tasks":     len(failedTasks),
	})

	return nil
}

// UpdateChunkStatusWithCracks updates a chunk's status and crack count
func (s *JobExecutionService) UpdateChunkStatusWithCracks(ctx context.Context, taskID uuid.UUID, crackCount int, detailedStatus string) error {
	debug.Log("Updating chunk status with crack information", map[string]interface{}{
		"task_id":         taskID,
		"crack_count":     crackCount,
		"detailed_status": detailedStatus,
	})

	err := s.jobTaskRepo.UpdateTaskWithCracks(ctx, taskID, crackCount, detailedStatus)
	if err != nil {
		return fmt.Errorf("failed to update task with cracks: %w", err)
	}

	return nil
}

// GetDynamicChunkSize calculates optimal chunk size based on agent benchmark data
func (s *JobExecutionService) GetDynamicChunkSize(ctx context.Context, agentID int, attackMode int, hashType int, defaultDurationSeconds int) (int64, error) {
	debug.Log("Calculating dynamic chunk size", map[string]interface{}{
		"agent_id":        agentID,
		"attack_mode":     attackMode,
		"hash_type":       hashType,
		"default_duration": defaultDurationSeconds,
	})

	// Get agent benchmark for this specific attack mode and hash type
	benchmark, err := s.benchmarkRepo.GetAgentBenchmark(ctx, agentID, models.AttackMode(attackMode), hashType)
	if err != nil {
		debug.Log("No benchmark found, using default chunk size", map[string]interface{}{
			"agent_id":    agentID,
			"attack_mode": attackMode,
			"hash_type":   hashType,
			"error":       err.Error(),
		})
		// Return a default chunk size (e.g., 1M keyspace)
		return 1000000, nil
	}

	// Calculate keyspace size for the default duration
	// keyspace = benchmark_speed * duration_seconds
	keyspaceSize := benchmark.Speed * int64(defaultDurationSeconds)

	debug.Log("Dynamic chunk size calculated", map[string]interface{}{
		"agent_id":        agentID,
		"benchmark_speed": benchmark.Speed,
		"duration":        defaultDurationSeconds,
		"keyspace_size":   keyspaceSize,
	})

	return keyspaceSize, nil
}

// resolveWordlistPath gets the actual file path for a wordlist ID
func (s *JobExecutionService) resolveWordlistPath(ctx context.Context, wordlistIDStr string) (string, error) {
	// Try to parse as integer ID first
	if wordlistID, err := strconv.Atoi(wordlistIDStr); err == nil {
		// Look up wordlist in database
		wordlists, err := s.fileRepo.GetWordlists(ctx, "")
		if err != nil {
			return "", fmt.Errorf("failed to get wordlists: %w", err)
		}
		
		for _, wl := range wordlists {
			if wl.ID == wordlistID {
				// The Name field now contains category/filename (e.g., "general/crackstation.txt")
				// We need to use just the filename without duplicating the category
				filename := wl.Name
				
				// If the Name already contains the category path, extract just the filename
				if strings.Contains(wl.Name, "/") {
					filename = filepath.Base(wl.Name)
				}
				
				// Build absolute path using the data directory
				var path string
				if wl.Category != "" {
					path = filepath.Join(s.dataDirectory, "wordlists", wl.Category, filename)
				} else {
					path = filepath.Join(s.dataDirectory, "wordlists", filename)
				}
				
				debug.Log("Resolved wordlist path", map[string]interface{}{
					"wordlist_id": wordlistID,
					"category":    wl.Category,
					"name_field":  wl.Name,
					"filename":    filename,
					"path":        path,
				})
				return path, nil
			}
		}
		return "", fmt.Errorf("wordlist with ID %d not found", wordlistID)
	}
	
	// If not a numeric ID, treat as a filename
	path := filepath.Join(s.dataDirectory, "wordlists", wordlistIDStr)
	debug.Log("Resolved wordlist path from string", map[string]interface{}{
		"wordlist_str": wordlistIDStr,
		"path":         path,
	})
	return path, nil
}

// resolveRulePath gets the actual file path for a rule ID
func (s *JobExecutionService) resolveRulePath(ctx context.Context, ruleIDStr string) (string, error) {
	// Try to parse as integer ID first
	if ruleID, err := strconv.Atoi(ruleIDStr); err == nil {
		// Look up rule in database
		rules, err := s.fileRepo.GetRules(ctx, "")
		if err != nil {
			return "", fmt.Errorf("failed to get rules: %w", err)
		}
		
		for _, rule := range rules {
			if rule.ID == ruleID {
				// The Name field now contains category/filename (e.g., "hashcat/wordlist_2f26acbe.txt")
				// We need to use just the filename without duplicating the category
				filename := rule.Name
				
				// If the Name already contains the category path, extract just the filename
				if strings.Contains(rule.Name, "/") {
					filename = filepath.Base(rule.Name)
				}
				
				// Build absolute path using the data directory
				var path string
				if rule.Category != "" {
					path = filepath.Join(s.dataDirectory, "rules", rule.Category, filename)
				} else {
					path = filepath.Join(s.dataDirectory, "rules", filename)
				}
				
				debug.Log("Resolved rule path", map[string]interface{}{
					"rule_id":    ruleID,
					"category":   rule.Category,
					"name_field": rule.Name,
					"filename":   filename,
					"path":       path,
				})
				return path, nil
			}
		}
		return "", fmt.Errorf("rule with ID %d not found", ruleID)
	}
	
	// If not a numeric ID, treat as a filename
	path := filepath.Join(s.dataDirectory, "rules", ruleIDStr)
	debug.Log("Resolved rule path from string", map[string]interface{}{
		"rule_str": ruleIDStr,
		"path":     path,
	})
	return path, nil
}
