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
	clientRepo           *repository.ClientRepository
	systemSettingsRepo   *repository.SystemSettingsRepository
	potfileService       *services.PotfileService
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
	clientRepo *repository.ClientRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	potfileService *services.PotfileService,
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
		clientRepo:           clientRepo,
		systemSettingsRepo:   systemSettingsRepo,
		potfileService:       potfileService,
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

	// Build wordlist and rule paths using job execution's self-contained configuration
	var wordlistPaths []string
	for _, wordlistIDStr := range jobExecution.WordlistIDs {
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
		for _, ruleIDStr := range jobExecution.RuleIDs {
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
	binaryVersion, err := s.binaryManager.GetVersion(ctx, int64(jobExecution.BinaryVersionID))
	if err != nil {
		return fmt.Errorf("failed to get binary version %d: %w", jobExecution.BinaryVersionID, err)
	}
	if binaryVersion == nil {
		return fmt.Errorf("binary version %d not found", jobExecution.BinaryVersionID)
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
		Mask:            jobExecution.Mask,
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

	// Update agent metadata to mark as busy AFTER successful send
	// This prevents agents from getting stuck in busy state if the assignment fails
	if agent.Metadata == nil {
		agent.Metadata = make(map[string]string)
	}
	agent.Metadata["busy_status"] = "true"
	agent.Metadata["current_task_id"] = task.ID.String()
	agent.Metadata["current_job_id"] = jobExecution.ID.String()
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("Failed to update agent metadata after task assignment: %v", err)
		// Don't fail the assignment, the agent is still running the task
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

	requestID := fmt.Sprintf("benchmark-%d-%d-%d-%d", agentID, hashType, attackMode, time.Now().Unix())
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
	for _, wordlistIDStr := range jobExecution.WordlistIDs {
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
	for _, ruleIDStr := range jobExecution.RuleIDs {
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
	binaryVersion, err := s.binaryManager.GetVersion(ctx, int64(jobExecution.BinaryVersionID))
	if err != nil {
		return fmt.Errorf("failed to get binary version %d: %w", jobExecution.BinaryVersionID, err)
	}
	if binaryVersion == nil {
		return fmt.Errorf("binary version %d not found", jobExecution.BinaryVersionID)
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
		"has_mask":        jobExecution.Mask != "",
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
		Mask:            jobExecution.Mask,
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
		debug.Warning("Received progress for non-existent task %d (ignoring): agent=%d, error=%v", progress.TaskID, agentID, err)
		// Don't return error - just ignore the update
		return nil
	}

	// Verify the task is assigned to this agent
	if task.AgentID == nil || *task.AgentID != agentID {
		expectedAgent := 0
		if task.AgentID != nil {
			expectedAgent = *task.AgentID
		}
		debug.Error("Progress from wrong agent: task=%d, expected=%d, actual=%d", progress.TaskID, expectedAgent, agentID)
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

	// Handle first progress update with actual effective keyspace from hashcat
	if progress.IsFirstUpdate && progress.TotalEffectiveKeyspace != nil && *progress.TotalEffectiveKeyspace > 0 {
		// This is the first update with hashcat's progress[1] - the true effective keyspace
		totalEffectiveKeyspace := *progress.TotalEffectiveKeyspace

		// Get job execution
		database := &db.DB{DB: s.db}
		jobExecRepo := repository.NewJobExecutionRepository(database)
		jobExec, err := jobExecRepo.GetByID(ctx, task.JobExecutionID)
		if err != nil {
			debug.Warning("Failed to get job execution for keyspace update: error=%v", err)
		} else {
			// Update task-level effective keyspace
			if task.IsRuleSplitTask && task.RuleStartIndex != nil && task.RuleEndIndex != nil && jobExec.BaseKeyspace != nil && *jobExec.BaseKeyspace > 0 {
				baseKeyspace := *jobExec.BaseKeyspace
				ruleStart := *task.RuleStartIndex
				ruleEnd := *task.RuleEndIndex

				// Calculate this task's effective keyspace range
				effectiveStart := baseKeyspace * int64(ruleStart)
				effectiveEnd := baseKeyspace * int64(ruleEnd)

				// Update task with actual effective keyspace
				err = s.jobTaskRepo.UpdateTaskEffectiveKeyspace(ctx, progress.TaskID, effectiveStart, effectiveEnd)
				if err != nil {
					debug.Warning("Failed to update task effective keyspace: task=%s, error=%v", progress.TaskID, err)
				} else {
					debug.Info("Updated task %s with actual effective keyspace: start=%d, end=%d, is_actual=true",
						progress.TaskID, effectiveStart, effectiveEnd)
				}

				// Update job-level effective keyspace if not yet accurate
				if !jobExec.IsAccurateKeyspace {
					// Calculate full job effective keyspace from this task's chunk
					// Per-rule keyspace = totalEffectiveKeyspace / rules_in_chunk
					rulesInChunk := ruleEnd - ruleStart
					if rulesInChunk > 0 {
						perRuleKeyspace := totalEffectiveKeyspace / int64(rulesInChunk)
						fullJobKeyspace := perRuleKeyspace * int64(jobExec.MultiplicationFactor)

						jobExec.EffectiveKeyspace = &fullJobKeyspace
						jobExec.IsAccurateKeyspace = true

						// Calculate avg_rule_multiplier
						if jobExec.BaseKeyspace != nil && *jobExec.BaseKeyspace > 0 {
							multiplier := float64(fullJobKeyspace) /
								float64(*jobExec.BaseKeyspace) /
								float64(jobExec.MultiplicationFactor)
							jobExec.AvgRuleMultiplier = &multiplier
						}

						// Update job in database
						err = s.jobExecutionService.UpdateKeyspaceInfo(ctx, jobExec)
						if err != nil {
							debug.Warning("Failed to update job keyspace info: job=%s, error=%v", jobExec.ID, err)
						} else {
							debug.Info("Updated job %s with accurate effective keyspace from first progress: %d (avg_rule_multiplier: %.5f)",
								jobExec.ID, fullJobKeyspace, *jobExec.AvgRuleMultiplier)
						}
					}
				}
			} else {
				// Non-rule-split task: effective keyspace matches the total
				effectiveStart := int64(0)
				effectiveEnd := totalEffectiveKeyspace

				err = s.jobTaskRepo.UpdateTaskEffectiveKeyspace(ctx, progress.TaskID, effectiveStart, effectiveEnd)
				if err != nil {
					debug.Warning("Failed to update task effective keyspace: task=%s, error=%v", progress.TaskID, err)
				} else {
					debug.Info("Updated task %s with actual effective keyspace: start=%d, end=%d, is_actual=true",
						progress.TaskID, effectiveStart, effectiveEnd)
				}

				// Update job-level effective keyspace if not yet accurate
				if !jobExec.IsAccurateKeyspace {
					jobExec.EffectiveKeyspace = &totalEffectiveKeyspace
					jobExec.IsAccurateKeyspace = true

					// Calculate avg_rule_multiplier
					if jobExec.BaseKeyspace != nil && *jobExec.BaseKeyspace > 0 && jobExec.MultiplicationFactor > 0 {
						multiplier := float64(totalEffectiveKeyspace) /
							float64(*jobExec.BaseKeyspace) /
							float64(jobExec.MultiplicationFactor)
						jobExec.AvgRuleMultiplier = &multiplier
					}

					// Update job in database
					err = s.jobExecutionService.UpdateKeyspaceInfo(ctx, jobExec)
					if err != nil {
						debug.Warning("Failed to update job keyspace info: job=%s, error=%v", jobExec.ID, err)
					} else {
						debug.Info("Updated job %s with accurate effective keyspace from first progress: %d",
							jobExec.ID, totalEffectiveKeyspace)
					}
				}
			}
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

		// Clear agent busy status
		if task.AgentID != nil {
			agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
			if err == nil && agent.Metadata != nil {
				agent.Metadata["busy_status"] = "false"
				delete(agent.Metadata, "current_task_id")
				delete(agent.Metadata, "current_job_id")
				if err := s.agentRepo.Update(ctx, agent); err != nil {
					debug.Error("Failed to clear agent busy status after task failure: %v", err)
				}
			}
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

		// Clear agent busy status
		if task.AgentID != nil {
			agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
			if err == nil && agent.Metadata != nil {
				agent.Metadata["busy_status"] = "false"
				delete(agent.Metadata, "current_task_id")
				delete(agent.Metadata, "current_job_id")
				if err := s.agentRepo.Update(ctx, agent); err != nil {
					debug.Error("Failed to clear agent busy status after task completion: %v", err)
				}
			}
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

	// Note: Hash rate metric recording removed here to prevent duplicate entries.
	// The metric is already recorded in job_scheduling_service.go with full device information.

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

		// Clear agent busy status
		if task.AgentID != nil {
			agent, err := s.agentRepo.GetByID(ctx, *task.AgentID)
			if err == nil && agent.Metadata != nil {
				agent.Metadata["busy_status"] = "false"
				delete(agent.Metadata, "current_task_id")
				delete(agent.Metadata, "current_job_id")
				if err := s.agentRepo.Update(ctx, agent); err != nil {
					debug.Error("Failed to clear agent busy status after task completion (keyspace): %v", err)
				}
			}
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

	// Handle total effective keyspace from hashcat progress[1]
	if result.TotalEffectiveKeyspace > 0 {
		// Find the job this benchmark is for
		// Get the next pending job for this agent (the one that requested the benchmark)
		job, err := s.jobExecutionService.GetNextJobWithWork(ctx)
		if err != nil || job == nil {
			debug.Warning("Could not find job for benchmark result from agent %d: %v", agentID, err)
			return nil // Don't fail - benchmark speed was still saved
		}

		// Use the JobExecution from the wrapper
		jobExec := &job.JobExecution

		// First benchmark for this job?
		if jobExec.EffectiveKeyspace == nil || !jobExec.IsAccurateKeyspace {
			// Set job-level effective keyspace from hashcat progress[1]
			jobExec.EffectiveKeyspace = &result.TotalEffectiveKeyspace
			jobExec.IsAccurateKeyspace = true

			// Calculate avg_rule_multiplier for future task estimates
			if jobExec.BaseKeyspace != nil && *jobExec.BaseKeyspace > 0 && jobExec.MultiplicationFactor > 0 {
				multiplier := float64(result.TotalEffectiveKeyspace) /
					float64(*jobExec.BaseKeyspace) /
					float64(jobExec.MultiplicationFactor)
				jobExec.AvgRuleMultiplier = &multiplier

				debug.Info("Job %s: Set accurate effective keyspace from hashcat: %d (avg_rule_multiplier: %.5f)",
					jobExec.ID, result.TotalEffectiveKeyspace, multiplier)
			} else {
				debug.Info("Job %s: Set accurate effective keyspace from hashcat: %d",
					jobExec.ID, result.TotalEffectiveKeyspace)
			}

			// Update job in database
			if err := s.jobExecutionService.UpdateKeyspaceInfo(ctx, jobExec); err != nil {
				debug.Error("Failed to update job keyspace info: %v", err)
				return fmt.Errorf("failed to update job keyspace info: %w", err)
			}
		} else {
			// Subsequent benchmark - validate consistency (should match job total)
			diff := result.TotalEffectiveKeyspace - *jobExec.EffectiveKeyspace
			if diff < 0 {
				diff = -diff // abs value
			}
			threshold := *jobExec.EffectiveKeyspace / 1000 // 0.1%

			if diff > threshold {
				debug.Warning("Agent %d benchmark differs from job total: observed=%d, expected=%d, diff=%d",
					agentID, result.TotalEffectiveKeyspace, *jobExec.EffectiveKeyspace, diff)
			} else {
				debug.Info("Agent %d benchmark validates job effective keyspace (diff=%d)", agentID, diff)
			}
		}
	}

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

		// Stage password for pot-file (non-blocking)
		// Check global potfile setting, client-level exclusion, AND per-hashlist exclusion
		if s.potfileService != nil && s.systemSettingsRepo != nil && s.hashlistRepo != nil && s.clientRepo != nil {
			potfileEnabled, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_enabled")
			if err == nil && potfileEnabled != nil && potfileEnabled.Value != nil && *potfileEnabled.Value == "true" {
				// Get hashlist to find client_id
				hashlist, err := s.hashlistRepo.GetByID(ctx, jobExecution.HashlistID)
				if err != nil {
					debug.Warning("Failed to get hashlist for potfile check: %v", err)
				} else {
					// Check if client has potfile excluded (if hashlist has a client)
					clientExcluded := false
					if hashlist.ClientID != uuid.Nil {
						clientExcluded, err = s.clientRepo.IsExcludedFromPotfile(ctx, hashlist.ClientID)
						if err != nil {
							debug.Warning("Failed to check client potfile exclusion: %v", err)
							clientExcluded = false // Default to not excluded on error
						}
					}

					if clientExcluded {
						debug.Info("Client %s is excluded from potfile, skipping password staging", hashlist.ClientID)
					} else {
						// Check if this specific hashlist is excluded from potfile
						hashlistExcluded, err := s.hashlistRepo.IsExcludedFromPotfile(ctx, jobExecution.HashlistID)
						if err != nil {
							debug.Warning("Failed to check hashlist potfile exclusion: %v", err)
						} else if hashlistExcluded {
							debug.Info("Hashlist %d is excluded from potfile, skipping password staging", jobExecution.HashlistID)
						} else {
							// All checks passed (global enabled, client not excluded, hashlist not excluded) - stage the password
							if err := s.potfileService.StagePassword(ctx, password, hashValue); err != nil {
								debug.Warning("Failed to stage password for pot-file: %v", err)
							} else {
								debug.Info("Successfully staged password for pot-file: hash=%s", hashValue)
							}
						}
					}
				}
			}
		}
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

// RecoverTask attempts to recover a task that was in reconnect_pending state
func (s *JobWebSocketIntegration) RecoverTask(ctx context.Context, taskID string, agentID int, keyspaceProcessed int64) error {
	debug.Log("Attempting to recover task", map[string]interface{}{
		"task_id":            taskID,
		"agent_id":           agentID,
		"keyspace_processed": keyspaceProcessed,
	})
	
	// Parse task ID as UUID
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Errorf("invalid task ID format: %w", err)
	}
	
	// Get the task from database
	task, err := s.jobTaskRepo.GetByID(ctx, taskUUID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}
	
	// Check task status and handle recovery appropriately
	switch task.Status {
	case models.JobTaskStatusRunning:
		// Task is already running, no recovery needed
		debug.Log("Task already running, no recovery needed", map[string]interface{}{
			"task_id": taskID,
			"status":  task.Status,
		})
		return nil
		
	case models.JobTaskStatusCompleted:
		// Task is already completed, agent shouldn't be running it
		debug.Log("Task already completed, agent should stop", map[string]interface{}{
			"task_id": taskID,
			"status":  task.Status,
		})
		// Return an error to trigger job_stop on the agent
		return fmt.Errorf("task %s is already completed", taskID)
		
	case models.JobTaskStatusReconnectPending, models.JobTaskStatusPending:
		// These states can be recovered
		debug.Log("Task can be recovered", map[string]interface{}{
			"task_id": taskID,
			"status":  task.Status,
		})
		// Continue with recovery below
		
	case models.JobTaskStatusFailed:
		// Check if task can be retried
		maxRetries := 3 // Get from settings
		if task.RetryCount < maxRetries {
			debug.Log("Failed task can be retried", map[string]interface{}{
				"task_id":     taskID,
				"status":      task.Status,
				"retry_count": task.RetryCount,
				"max_retries": maxRetries,
			})
			// Continue with recovery below
		} else {
			return fmt.Errorf("task %s has exceeded maximum retries (%d)", taskID, maxRetries)
		}
		
	default:
		// Other states (cancelled, etc.) cannot be recovered
		return fmt.Errorf("task %s cannot be recovered from state: %s", taskID, task.Status)
	}
	
	// Update task status back to running and reassign to the agent
	err = s.jobTaskRepo.UpdateStatus(ctx, taskUUID, models.JobTaskStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	// Update task assignment to the reconnected agent
	task.AgentID = &agentID
	task.Status = models.JobTaskStatusRunning
	task.DetailedStatus = "running" // Ensure detailed_status matches the status for constraint
	if keyspaceProcessed > 0 {
		task.KeyspaceProcessed = keyspaceProcessed
	}
	
	err = s.jobTaskRepo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to update task assignment: %w", err)
	}
	
	debug.Log("Successfully recovered task", map[string]interface{}{
		"task_id":  taskID,
		"agent_id": agentID,
		"job_id":   task.JobExecutionID,
	})
	
	// Ensure the job remains in running state
	// Wrap sql.DB in custom DB type
	database := &db.DB{DB: s.db}
	jobExecRepo := repository.NewJobExecutionRepository(database)
	err = jobExecRepo.UpdateStatus(ctx, task.JobExecutionID, models.JobExecutionStatusRunning)
	if err != nil {
		// Log but don't fail - task recovery is more important
		debug.Log("Failed to update job status during task recovery", map[string]interface{}{
			"job_id": task.JobExecutionID,
			"error":  err.Error(),
		})
	}
	
	return nil
}

// HandleAgentDisconnection marks tasks as reconnect_pending when an agent disconnects
func (s *JobWebSocketIntegration) HandleAgentDisconnection(ctx context.Context, agentID int) error {
	debug.Log("Handling agent disconnection", map[string]interface{}{
		"agent_id": agentID,
	})
	
	// Find all running or assigned tasks for this agent
	// Wrap sql.DB in custom DB type
	database := &db.DB{DB: s.db}
	taskRepo := repository.NewJobTaskRepository(database)
	
	// Get task IDs that are currently running or assigned to this agent
	taskIDs, err := taskRepo.GetTasksByAgentAndStatus(ctx, agentID, models.JobTaskStatusRunning)
	if err != nil {
		debug.Log("Failed to get running tasks for disconnected agent", map[string]interface{}{
			"agent_id": agentID,
			"error":    err.Error(),
		})
	}
	
	// Also get assigned tasks
	assignedTaskIDs, err := taskRepo.GetTasksByAgentAndStatus(ctx, agentID, models.JobTaskStatusAssigned)
	if err != nil {
		debug.Log("Failed to get assigned tasks for disconnected agent", map[string]interface{}{
			"agent_id": agentID,
			"error":    err.Error(),
		})
	}
	
	// Combine both lists
	if assignedTaskIDs != nil {
		taskIDs = append(taskIDs, assignedTaskIDs...)
	}
	
	// Get full task objects and mark each as reconnect_pending
	var tasks []models.JobTask
	for _, taskID := range taskIDs {
		// Get the full task object
		task, err := taskRepo.GetByID(ctx, taskID)
		if err != nil || task == nil {
			debug.Log("Failed to get task details", map[string]interface{}{
				"task_id": taskID,
				"error":   err,
			})
			continue
		}
		
		debug.Log("Marking task as reconnect_pending due to agent disconnection", map[string]interface{}{
			"task_id":  taskID,
			"agent_id": agentID,
			"job_id":   task.JobExecutionID,
		})
		
		// Update task status to reconnect_pending
		err = taskRepo.UpdateStatus(ctx, taskID, models.JobTaskStatusReconnectPending)
		if err != nil {
			debug.Log("Failed to mark task as reconnect_pending", map[string]interface{}{
				"task_id": taskID,
				"error":   err.Error(),
			})
			continue
		}
		
		// Clear the agent_id from the task so it can be reassigned
		task.AgentID = nil
		task.Status = models.JobTaskStatusReconnectPending
		err = taskRepo.Update(ctx, task)
		if err != nil {
			debug.Log("Failed to clear agent_id from task", map[string]interface{}{
				"task_id": taskID,
				"error":   err.Error(),
			})
		}
		
		tasks = append(tasks, *task)
	}
	
	if len(tasks) > 0 {
		debug.Log("Successfully marked tasks as reconnect_pending", map[string]interface{}{
			"agent_id":    agentID,
			"task_count":  len(tasks),
		})
		
		// Start a timer to handle grace period expiration (2 minutes)
		go s.handleReconnectGracePeriod(ctx, tasks, agentID)
	}
	
	return nil
}

// HandleAgentReconnectionWithNoTask handles when an agent reconnects but reports no running task
// It finds all reconnect_pending tasks assigned to this agent and resets them for retry
func (s *JobWebSocketIntegration) HandleAgentReconnectionWithNoTask(ctx context.Context, agentID int) (int, error) {
	debug.Log("Handling agent reconnection with no running task", map[string]interface{}{
		"agent_id": agentID,
	})
	
	// Get all reconnect_pending tasks for this agent
	reconnectTasks, err := s.jobTaskRepo.GetReconnectPendingTasksByAgent(ctx, agentID)
	if err != nil {
		debug.Log("Failed to get reconnect_pending tasks for agent", map[string]interface{}{
			"agent_id": agentID,
			"error":    err.Error(),
		})
		return 0, fmt.Errorf("failed to get reconnect_pending tasks: %w", err)
	}
	
	if len(reconnectTasks) == 0 {
		debug.Log("No reconnect_pending tasks found for agent", map[string]interface{}{
			"agent_id": agentID,
		})
		return 0, nil
	}
	
	debug.Log("Found reconnect_pending tasks to reset", map[string]interface{}{
		"agent_id":   agentID,
		"task_count": len(reconnectTasks),
	})
	
	// Get max retry attempts from settings
	maxRetries := 3
	retrySetting, err := s.systemSettingsRepo.GetSetting(ctx, "max_chunk_retry_attempts")
	if err == nil && retrySetting.Value != nil {
		if retries, err := strconv.Atoi(*retrySetting.Value); err == nil {
			maxRetries = retries
		}
	}
	
	resetCount := 0
	failedCount := 0
	
	for _, task := range reconnectTasks {
		// Check if task can be retried
		if task.RetryCount < maxRetries {
			// Reset task for retry
			err := s.jobTaskRepo.ResetTaskForRetry(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to reset task for retry", map[string]interface{}{
					"task_id":  task.ID,
					"agent_id": agentID,
					"error":    err.Error(),
				})
				continue
			}
			
			debug.Log("Task reset for retry after agent reconnection", map[string]interface{}{
				"task_id":      task.ID,
				"agent_id":     agentID,
				"retry_count":  task.RetryCount + 1,
				"max_retries":  maxRetries,
			})
			resetCount++
		} else {
			// Mark as permanently failed after all retries exhausted
			errorMsg := fmt.Sprintf("Agent %d reconnected without task after %d retry attempts", agentID, task.RetryCount)
			err := s.jobTaskRepo.UpdateTaskError(ctx, task.ID, errorMsg)
			if err != nil {
				debug.Log("Failed to mark task as failed", map[string]interface{}{
					"task_id":  task.ID,
					"agent_id": agentID,
					"error":    err.Error(),
				})
				continue
			}
			
			debug.Log("Task permanently failed after max retries", map[string]interface{}{
				"task_id":     task.ID,
				"agent_id":    agentID,
				"retry_count": task.RetryCount,
			})
			failedCount++
		}
	}
	
	debug.Log("Completed processing reconnect_pending tasks for agent", map[string]interface{}{
		"agent_id":     agentID,
		"total_tasks":  len(reconnectTasks),
		"reset_count":  resetCount,
		"failed_count": failedCount,
	})
	
	// Check if affected jobs need status update
	jobIDs := make(map[uuid.UUID]bool)
	for _, task := range reconnectTasks {
		jobIDs[task.JobExecutionID] = true
	}
	
	for jobID := range jobIDs {
		// Check if any tasks are still active for this job
		allTasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
		if err != nil {
			debug.Log("Failed to check job tasks", map[string]interface{}{
				"job_id": jobID,
				"error":  err.Error(),
			})
			continue
		}
		
		hasActiveTasks := false
		for _, task := range allTasks {
			if task.Status == models.JobTaskStatusRunning || 
			   task.Status == models.JobTaskStatusReconnectPending ||
			   task.Status == models.JobTaskStatusAssigned {
				hasActiveTasks = true
				break
			}
		}
		
		// If no active tasks remain and we have pending tasks, ensure job is in pending state
		if !hasActiveTasks {
			hasPendingTasks := false
			for _, task := range allTasks {
				if task.Status == models.JobTaskStatusPending {
					hasPendingTasks = true
					break
				}
			}
			
			if hasPendingTasks {
				// Ensure job is in pending state for rescheduling
				// Use jobExecutionRepo from the service
				database := &db.DB{DB: s.db}
				jobExecutionRepo := repository.NewJobExecutionRepository(database)
				err := jobExecutionRepo.UpdateStatus(ctx, jobID, models.JobExecutionStatusPending)
				if err != nil {
					debug.Log("Failed to update job status to pending", map[string]interface{}{
						"job_id": jobID,
						"error":  err.Error(),
					})
				} else {
					debug.Log("Job marked as pending for rescheduling", map[string]interface{}{
						"job_id": jobID,
					})
				}
			}
		}
	}
	
	return resetCount, nil
}

// handleReconnectGracePeriod waits for the grace period and then marks tasks as failed if not recovered
func (s *JobWebSocketIntegration) handleReconnectGracePeriod(ctx context.Context, tasks []models.JobTask, agentID int) {
	gracePeriod := 2 * time.Minute
	debug.Log("Starting reconnect grace period timer", map[string]interface{}{
		"agent_id":      agentID,
		"task_count":    len(tasks),
		"grace_period":  gracePeriod.String(),
	})
	
	time.Sleep(gracePeriod)
	
	debug.Log("Grace period expired, checking tasks", map[string]interface{}{
		"agent_id": agentID,
	})
	
	// Wrap sql.DB in custom DB type
	database := &db.DB{DB: s.db}
	taskRepo := repository.NewJobTaskRepository(database)
	
	for _, task := range tasks {
		// Check if task is still in reconnect_pending state
		currentTask, err := taskRepo.GetByID(ctx, task.ID)
		if err != nil {
			debug.Log("Failed to get task status after grace period", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			continue
		}
		
		if currentTask != nil && currentTask.Status == models.JobTaskStatusReconnectPending {
			debug.Log("Task still in reconnect_pending after grace period, marking as pending for reassignment", map[string]interface{}{
				"task_id": task.ID,
			})
			
			// Mark task as pending so it can be reassigned to another agent
			err = taskRepo.UpdateStatus(ctx, task.ID, models.JobTaskStatusPending)
			if err != nil {
				debug.Log("Failed to mark task as pending after grace period", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
			}
		}
	}
}
