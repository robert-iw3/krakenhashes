package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/rule"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	wsservice "github.com/ZerkerEOD/krakenhashes/backend/internal/services/websocket"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

// JobWebSocketIntegration handles the integration between job scheduling and WebSocket communication
type JobWebSocketIntegration struct {
	wsHandler interface {
		SendMessage(agentID int, msg *wsservice.Message) error
	}
	jobSchedulingService *services.JobSchedulingService
	jobExecutionService  *services.JobExecutionService
	hashlistSyncService  *services.HashlistSyncService
	benchmarkRepo        *repository.BenchmarkRepository
	presetJobRepo        repository.PresetJobRepository
	hashlistRepo         *repository.HashListRepository
	hashRepo             *repository.HashRepository
	jobTaskRepo          *repository.JobTaskRepository
	agentRepo            *repository.AgentRepository
	deviceRepo           *repository.AgentDeviceRepository
	systemSettingsRepo   *repository.SystemSettingsRepository
	db                   *sql.DB
	wordlistManager      wordlist.Manager
	ruleManager          rule.Manager
	binaryManager        binary.Manager

	// Progress tracking
	progressMutex   sync.RWMutex
	taskProgressMap map[string]*models.JobProgress // TaskID -> Progress
}

// NewJobWebSocketIntegration creates a new job WebSocket integration service
func NewJobWebSocketIntegration(
	wsHandler interface {
		SendMessage(agentID int, msg *wsservice.Message) error
	},
	jobSchedulingService *services.JobSchedulingService,
	jobExecutionService *services.JobExecutionService,
	hashlistSyncService *services.HashlistSyncService,
	benchmarkRepo *repository.BenchmarkRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	hashRepo *repository.HashRepository,
	jobTaskRepo *repository.JobTaskRepository,
	agentRepo *repository.AgentRepository,
	deviceRepo *repository.AgentDeviceRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	db *sql.DB,
	wordlistManager wordlist.Manager,
	ruleManager rule.Manager,
	binaryManager binary.Manager,
) *JobWebSocketIntegration {
	return &JobWebSocketIntegration{
		wsHandler:            wsHandler,
		jobSchedulingService: jobSchedulingService,
		jobExecutionService:  jobExecutionService,
		hashlistSyncService:  hashlistSyncService,
		benchmarkRepo:        benchmarkRepo,
		presetJobRepo:        presetJobRepo,
		hashlistRepo:         hashlistRepo,
		hashRepo:             hashRepo,
		jobTaskRepo:          jobTaskRepo,
		agentRepo:            agentRepo,
		deviceRepo:           deviceRepo,
		systemSettingsRepo:   systemSettingsRepo,
		db:                   db,
		wordlistManager:      wordlistManager,
		ruleManager:          ruleManager,
		binaryManager:        binaryManager,
		taskProgressMap:      make(map[string]*models.JobProgress),
	}
}

// SendJobAssignment sends a job task assignment to an agent via WebSocket
func (s *JobWebSocketIntegration) SendJobAssignment(ctx context.Context, task *models.JobTask, jobExecution *models.JobExecution) error {
	debug.Log("Sending job assignment to agent", map[string]interface{}{
		"task_id":  task.ID,
		"agent_id": task.AgentID,
		"job_id":   jobExecution.ID,
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
	if task.AgentID == nil {
		return fmt.Errorf("task has no agent assigned")
	}
	agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Build wordlist and rule paths
	var wordlistPaths []string
	for _, wordlistIDStr := range presetJob.WordlistIDs {
		// Convert string ID to int
		wordlistID, err := strconv.Atoi(wordlistIDStr)
		if err != nil {
			return fmt.Errorf("invalid wordlist ID %s: %w", wordlistIDStr, err)
		}

		// Look up the actual wordlist file path
		wordlist, err := s.wordlistManager.GetWordlist(ctx, wordlistID)
		if err != nil {
			return fmt.Errorf("failed to get wordlist %d: %w", wordlistID, err)
		}
		if wordlist == nil {
			return fmt.Errorf("wordlist %d not found", wordlistID)
		}

		// Use the actual file path from the database
		wordlistPath := fmt.Sprintf("wordlists/%s", wordlist.FileName)
		wordlistPaths = append(wordlistPaths, wordlistPath)
	}

	var rulePaths []string
	// Check if this is a rule split task with a chunk file
	if task.IsRuleSplitTask && task.RuleChunkPath != nil && *task.RuleChunkPath != "" {
		// Extract job directory from the chunk path
		pathParts := strings.Split(*task.RuleChunkPath, string(filepath.Separator))
		var jobDirName string
		chunkFilename := filepath.Base(*task.RuleChunkPath)

		// Find the job directory name
		for i, part := range pathParts {
			if strings.HasPrefix(part, "job_") && i < len(pathParts)-1 {
				jobDirName = part
				break
			}
		}

		// Create the rule path with job directory
		var rulePath string
		if jobDirName != "" {
			rulePath = fmt.Sprintf("rules/chunks/%s/%s", jobDirName, chunkFilename)
		} else {
			// Fallback to just chunk filename
			rulePath = fmt.Sprintf("rules/chunks/%s", chunkFilename)
		}
		rulePaths = append(rulePaths, rulePath)

		debug.Log("Using rule chunk for task", map[string]interface{}{
			"task_id":    task.ID,
			"chunk_path": *task.RuleChunkPath,
			"agent_path": rulePath,
			"job_dir":    jobDirName,
		})
	} else {
		// Standard rule processing
		for _, ruleIDStr := range presetJob.RuleIDs {
			// Convert string ID to int
			ruleID, err := strconv.Atoi(ruleIDStr)
			if err != nil {
				return fmt.Errorf("invalid rule ID %s: %w", ruleIDStr, err)
			}

			// Look up the actual rule file path
			rule, err := s.ruleManager.GetRule(ctx, ruleID)
			if err != nil {
				return fmt.Errorf("failed to get rule %d: %w", ruleID, err)
			}
			if rule == nil {
				return fmt.Errorf("rule %d not found", ruleID)
			}

			// Use the actual file path from the database
			rulePath := fmt.Sprintf("rules/%s", rule.FileName)
			rulePaths = append(rulePaths, rulePath)
		}
	}

	// Get binary path from binary version
	binaryVersion, err := s.binaryManager.GetVersion(ctx, int64(presetJob.BinaryVersionID))
	if err != nil {
		return fmt.Errorf("failed to get binary version %d: %w", presetJob.BinaryVersionID, err)
	}
	if binaryVersion == nil {
		return fmt.Errorf("binary version %d not found", presetJob.BinaryVersionID)
	}

	// Use the actual binary path - the ID is used as the directory name
	binaryPath := fmt.Sprintf("binaries/%d", binaryVersion.ID)

	// Get report interval from settings or use default
	reportInterval := 5 // Default 5 seconds
	if val, err := s.jobExecutionService.GetSystemSetting(ctx, "progress_reporting_interval"); err == nil {
		reportInterval = val
	}

	// Get enabled devices for the agent
	var enabledDeviceIDs []int
	if task.AgentID != nil {
		devices, err := s.deviceRepo.GetByAgentID(*task.AgentID)
		if err != nil {
			debug.Error("Failed to get agent devices: %v", err)
			// Continue without device specification
		} else {
			// Only include device IDs if some devices are disabled
			hasDisabledDevice := false
			for _, device := range devices {
				if !device.Enabled {
					hasDisabledDevice = true
				} else {
					enabledDeviceIDs = append(enabledDeviceIDs, device.DeviceID)
				}
			}
			// If all devices are enabled, don't include the device list
			if !hasDisabledDevice {
				enabledDeviceIDs = nil
			}
		}
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
		OutputFormat:    "3",                   // hash:plain format
		ExtraParameters: agent.ExtraParameters, // Agent-specific hashcat parameters
		EnabledDevices:  enabledDeviceIDs,      // Only populated if some devices are disabled
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

	// Update task status to assigned BEFORE sending via WebSocket
	// This ensures the task is marked as assigned even if WebSocket fails
	err = s.jobTaskRepo.UpdateStatus(ctx, task.ID, models.JobTaskStatusAssigned)
	if err != nil {
		return fmt.Errorf("failed to update task status to assigned: %w", err)
	}

	// Send via WebSocket
	err = s.wsHandler.SendMessage(agent.ID, msg)
	if err != nil {
		// Revert task status back to pending since we couldn't send it
		revertErr := s.jobTaskRepo.UpdateStatus(ctx, task.ID, models.JobTaskStatusPending)
		if revertErr != nil {
			debug.Error("Failed to revert task status after WebSocket error: %v", revertErr)
		}
		return fmt.Errorf("failed to send task assignment via WebSocket: %w", err)
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
	if task.AgentID == nil {
		return fmt.Errorf("task has no agent assigned")
	}
	agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
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
// SendForceCleanup sends a force cleanup command to an agent
func (s *JobWebSocketIntegration) SendForceCleanup(ctx context.Context, agentID int) error {
	debug.Log("Sending force cleanup command to agent", map[string]interface{}{
		"agent_id": agentID,
	})

	// Create the force cleanup message
	msg := &wsservice.Message{
		Type: wsservice.TypeForceCleanup,
		// No payload needed for force cleanup
		Payload: json.RawMessage("{}"),
	}

	// Send the message to the agent
	if err := s.wsHandler.SendMessage(agentID, msg); err != nil {
		return fmt.Errorf("failed to send force cleanup: %w", err)
	}

	debug.Log("Force cleanup command sent successfully", map[string]interface{}{
		"agent_id": agentID,
	})

	return nil
}

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

// RequestAgentBenchmark implements the JobWebSocketIntegration interface for requesting benchmarks
func (s *JobWebSocketIntegration) RequestAgentBenchmark(ctx context.Context, agentID int, jobExecution *models.JobExecution) error {
	// Get preset job details to find binary version and attack configuration
	presetJob, err := s.presetJobRepo.GetByID(ctx, jobExecution.PresetJobID)
	if err != nil {
		return fmt.Errorf("failed to get preset job: %w", err)
	}

	// Get hashlist to get hash type
	hashlist, err := s.hashlistRepo.GetByID(ctx, jobExecution.HashlistID)
	if err != nil {
		return fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Get agent details
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Build wordlist and rule paths for a more accurate benchmark
	var wordlistPaths []string
	for _, wordlistIDStr := range presetJob.WordlistIDs {
		// Convert string ID to int
		wordlistID, err := strconv.Atoi(wordlistIDStr)
		if err != nil {
			continue // Skip invalid IDs
		}

		// Look up the actual wordlist file path
		wordlist, err := s.wordlistManager.GetWordlist(ctx, wordlistID)
		if err != nil || wordlist == nil {
			continue // Skip missing wordlists
		}

		// Use the actual file path from the database
		wordlistPath := fmt.Sprintf("wordlists/%s", wordlist.FileName)
		wordlistPaths = append(wordlistPaths, wordlistPath)
	}

	var rulePaths []string
	for _, ruleIDStr := range presetJob.RuleIDs {
		// Convert string ID to int
		ruleID, err := strconv.Atoi(ruleIDStr)
		if err != nil {
			continue // Skip invalid IDs
		}

		// Look up the actual rule file path
		rule, err := s.ruleManager.GetRule(ctx, ruleID)
		if err != nil || rule == nil {
			continue // Skip missing rules
		}

		// Use the actual file path from the database
		rulePath := fmt.Sprintf("rules/%s", rule.FileName)
		rulePaths = append(rulePaths, rulePath)
	}

	// Get binary path from binary version
	binaryVersion, err := s.binaryManager.GetVersion(ctx, int64(presetJob.BinaryVersionID))
	if err != nil {
		return fmt.Errorf("failed to get binary version %d: %w", presetJob.BinaryVersionID, err)
	}
	if binaryVersion == nil {
		return fmt.Errorf("binary version %d not found", presetJob.BinaryVersionID)
	}

	// Use the actual binary path - the ID is used as the directory name
	binaryPath := fmt.Sprintf("binaries/%d", binaryVersion.ID)

	// Get enabled devices for the agent
	var enabledDeviceIDs []int
	devices, err := s.deviceRepo.GetByAgentID(agentID)
	if err != nil {
		debug.Error("Failed to get agent devices for benchmark: %v", err)
		// Continue without device specification
	} else {
		// Only include device IDs if some devices are disabled
		hasDisabledDevice := false
		for _, device := range devices {
			if !device.Enabled {
				hasDisabledDevice = true
			} else {
				enabledDeviceIDs = append(enabledDeviceIDs, device.DeviceID)
			}
		}
		// If all devices are enabled, don't include the device list
		if !hasDisabledDevice {
			enabledDeviceIDs = nil
		}
	}

	requestID := fmt.Sprintf("benchmark-%d-%d-%d-%d", agentID, hashlist.HashTypeID, jobExecution.AttackMode, time.Now().Unix())

	debug.Log("Sending enhanced benchmark request to agent", map[string]interface{}{
		"agent_id":        agentID,
		"hash_type":       hashlist.HashTypeID,
		"attack_mode":     jobExecution.AttackMode,
		"request_id":      requestID,
		"wordlist_count":  len(wordlistPaths),
		"rule_count":      len(rulePaths),
		"has_mask":        presetJob.Mask != "",
		"enabled_devices": enabledDeviceIDs,
	})

	// Get speedtest timeout from system settings
	speedtestTimeout := 180 // Default to 3 minutes
	if s.systemSettingsRepo != nil {
		if setting, err := s.systemSettingsRepo.GetSetting(ctx, "speedtest_timeout_seconds"); err == nil && setting.Value != nil {
			if timeout, err := strconv.Atoi(*setting.Value); err == nil && timeout > 0 {
				speedtestTimeout = timeout
			}
		}
	}

	// Create enhanced benchmark request payload with job-specific configuration
	benchmarkReq := wsservice.BenchmarkRequestPayload{
		RequestID:       requestID,
		TaskID:          fmt.Sprintf("benchmark-%s-%d", jobExecution.ID, time.Now().Unix()), // Generate a task ID for the benchmark
		HashType:        hashlist.HashTypeID,
		AttackMode:      int(jobExecution.AttackMode),
		BinaryPath:      binaryPath,
		HashlistID:      jobExecution.HashlistID,
		HashlistPath:    fmt.Sprintf("hashlists/%d.hash", jobExecution.HashlistID),
		WordlistPaths:   wordlistPaths,
		RulePaths:       rulePaths,
		Mask:            presetJob.Mask,
		TestDuration:    30,                    // 30-second benchmark for accuracy
		TimeoutDuration: speedtestTimeout,      // Configurable timeout for speedtest
		ExtraParameters: agent.ExtraParameters, // Agent-specific hashcat parameters
		EnabledDevices:  enabledDeviceIDs,      // Only populated if some devices are disabled
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

	debug.Log("Enhanced benchmark request sent successfully", map[string]interface{}{
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
		"effective_progress": progress.EffectiveProgress,
		"progress_percent":   progress.ProgressPercent,
		"hash_rate":          progress.HashRate,
	})

	// Validate task exists before processing
	task, err := s.jobTaskRepo.GetByID(ctx, progress.TaskID)
	if err != nil {
		// Log and ignore progress updates for non-existent tasks (could be orphaned)
		debug.Warning("Received progress for non-existent task (ignoring)", map[string]interface{}{
			"task_id":  progress.TaskID,
			"agent_id": agentID,
			"error":    err.Error(),
		})
		// Don't return error - just ignore the update
		return nil
	}

	// Verify the task is assigned to this agent
	if task.AgentID == nil || *task.AgentID != agentID {
		debug.Error("Progress from wrong agent", map[string]interface{}{
			"task_id":        progress.TaskID,
			"expected_agent": task.AgentID,
			"actual_agent":   agentID,
		})
		return fmt.Errorf("task not assigned to this agent")
	}

	// Update task status to running if it's still assigned
	if task.Status == models.JobTaskStatusAssigned {
		// Use StartTask to update both status and started_at timestamp
		err = s.jobTaskRepo.StartTask(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to start task", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
			// Fallback to just updating status
			err = s.jobTaskRepo.UpdateStatus(ctx, progress.TaskID, models.JobTaskStatusRunning)
			if err != nil {
				debug.Log("Failed to update task status to running", map[string]interface{}{
					"task_id": progress.TaskID,
					"error":   err.Error(),
				})
			}
		} else {
			debug.Log("Started task", map[string]interface{}{
				"task_id": progress.TaskID,
			})
		}
	}

	// Store progress in memory
	s.progressMutex.Lock()
	s.taskProgressMap[progress.TaskID.String()] = progress
	s.progressMutex.Unlock()

	// Check if this is a failure update
	if progress.Status == "failed" && progress.ErrorMessage != "" {
		debug.Log("Task failed with error", map[string]interface{}{
			"task_id": progress.TaskID,
			"error":   progress.ErrorMessage,
		})

		// Update task status to failed
		err := s.jobTaskRepo.UpdateTaskError(ctx, progress.TaskID, progress.ErrorMessage)
		if err != nil {
			debug.Error("Failed to update task error: %v", err)
		}

		// Update job execution status to failed
		// Wrap sql.DB in custom DB type
		database := &db.DB{DB: s.db}
		jobExecRepo := repository.NewJobExecutionRepository(database)
		if err := jobExecRepo.UpdateStatus(ctx, task.JobExecutionID, models.JobExecutionStatusFailed); err != nil {
			debug.Error("Failed to update job execution status: %v", err)
		}
		if err := jobExecRepo.UpdateErrorMessage(ctx, task.JobExecutionID, progress.ErrorMessage); err != nil {
			debug.Error("Failed to update job execution error message: %v", err)
		}

		// Handle task failure cleanup
		err = s.jobExecutionService.HandleTaskCompletion(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to handle failed task cleanup", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
		}

		return nil
	}

	// Check if this is a completion update
	if progress.Status == "completed" {
		debug.Log("Task completed", map[string]interface{}{
			"task_id":          progress.TaskID,
			"progress_percent": progress.ProgressPercent,
		})

		// Update the final progress first
		err := s.jobSchedulingService.ProcessTaskProgress(ctx, progress.TaskID, progress)
		if err != nil {
			debug.Error("Failed to process final task progress: %v", err)
		}

		// Then mark task as complete
		err = s.jobTaskRepo.CompleteTask(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to mark task as complete", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
		}

		// Reset consecutive failure counters on success
		err = s.jobSchedulingService.HandleTaskSuccess(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to handle task success", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
		}

		// Handle task completion cleanup
		err = s.jobExecutionService.HandleTaskCompletion(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to handle task completion", map[string]interface{}{
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

		return nil
	}

	// Forward to job scheduling service for normal progress updates
	err = s.jobSchedulingService.ProcessTaskProgress(ctx, progress.TaskID, progress)
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

	// Check if task is complete based on keyspace
	if progress.KeyspaceProcessed >= (task.KeyspaceEnd - task.KeyspaceStart) {
		// Task is complete
		err = s.jobTaskRepo.CompleteTask(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to mark task as complete", map[string]interface{}{
				"task_id": progress.TaskID,
				"error":   err.Error(),
			})
		}

		// Handle task completion cleanup
		err = s.jobExecutionService.HandleTaskCompletion(ctx, progress.TaskID)
		if err != nil {
			debug.Log("Failed to handle task completion", map[string]interface{}{
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
func (s *JobWebSocketIntegration) processCrackedHashes(ctx context.Context, taskID uuid.UUID, crackedHashes []models.CrackedHash) error {
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

	// Start a transaction for updating cracked hashes
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	var crackedCount int
	crackedAt := time.Now()

	// Process each cracked hash
	for _, crackedEntry := range crackedHashes {
		hashValue := crackedEntry.Hash
		password := crackedEntry.Plain
		crackPos := crackedEntry.CrackPos

		// Find the hash in the database
		hashes, err := s.hashRepo.GetByHashValues(ctx, []string{hashValue})
		if err != nil {
			return fmt.Errorf("failed to find hash: %w", err)
		}

		if len(hashes) == 0 {
			debug.Log("Hash not found in hashlist", map[string]interface{}{
				"hash_value":  hashValue,
				"hashlist_id": jobExecution.HashlistID,
			})
			continue
		}

		// For now, we'll use the first hash found
		// In a production system, we'd need to verify this hash belongs to the correct hashlist
		// by checking the hashlist_hashes junction table
		hash := hashes[0]

		// Check if hash is already cracked to prevent double counting
		if hash.IsCracked {
			debug.Log("Hash already cracked, skipping", map[string]interface{}{
				"hash_id":     hash.ID,
				"hash_value":  hashValue,
				"hashlist_id": jobExecution.HashlistID,
			})
			continue
		}

		// Update crack status
		err = s.hashRepo.UpdateCrackStatus(tx, hash.ID, password, crackedAt, nil)
		if err != nil {
			debug.Log("Failed to update crack status", map[string]interface{}{
				"hash_id": hash.ID,
				"error":   err.Error(),
			})
			continue
		}

		crackedCount++
		debug.Log("Successfully cracked hash", map[string]interface{}{
			"hash_id":     hash.ID,
			"hash_value":  hashValue,
			"hashlist_id": jobExecution.HashlistID,
			"crack_pos":   crackPos,
			"password":    password,
		})
	}

	// Update hashlist cracked count
	if crackedCount > 0 {
		err = s.hashlistRepo.IncrementCrackedCount(ctx, jobExecution.HashlistID, crackedCount)
		if err != nil {
			debug.Log("Failed to update hashlist cracked count", map[string]interface{}{
				"hashlist_id": jobExecution.HashlistID,
				"error":       err.Error(),
			})
		}

		// Update job task crack count
		err = s.jobTaskRepo.UpdateCrackCount(ctx, taskID, crackedCount)
		if err != nil {
			debug.Log("Failed to update job task crack count", map[string]interface{}{
				"task_id": taskID,
				"error":   err.Error(),
			})
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update hashlist file to remove cracked hashes
	// Convert CrackedHash array to string array for backward compatibility
	var crackedHashStrings []string
	for _, cracked := range crackedHashes {
		// Format as hash:plain for hashlist sync
		crackedHashStrings = append(crackedHashStrings, fmt.Sprintf("%s:%s", cracked.Hash, cracked.Plain))
	}

	err = s.hashlistSyncService.UpdateHashlistAfterCracks(ctx, jobExecution.HashlistID, crackedHashStrings)
	if err != nil {
		debug.Log("Failed to update hashlist file after cracks", map[string]interface{}{
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
