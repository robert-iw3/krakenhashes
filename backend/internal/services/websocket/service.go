package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// JobHandler interface for handling job-related WebSocket messages
type JobHandler interface {
	ProcessJobProgress(ctx context.Context, agentID int, payload json.RawMessage) error
	ProcessBenchmarkResult(ctx context.Context, agentID int, payload json.RawMessage) error
	RecoverTask(ctx context.Context, taskID string, agentID int, keyspaceProcessed int64) error
	HandleAgentReconnectionWithNoTask(ctx context.Context, agentID int) (int, error)
}

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Agent -> Server messages
	TypeHeartbeat       MessageType = "heartbeat"
	TypeTaskStatus      MessageType = "task_status"
	TypeJobProgress     MessageType = "job_progress"
	TypeBenchmarkResult MessageType = "benchmark_result"
	TypeAgentStatus     MessageType = "agent_status"
	TypeErrorReport     MessageType = "error_report"
	TypeHardwareInfo    MessageType = "hardware_info"
	TypeSyncResponse    MessageType = "file_sync_response"
	TypeSyncStatus      MessageType = "file_sync_status"
	TypeHashcatOutput    MessageType = "hashcat_output"
	TypeDeviceDetection  MessageType = "device_detection"
	TypeDeviceUpdate     MessageType = "device_update"
	TypeBufferedMessages MessageType = "buffered_messages"
	TypeCurrentTaskStatus MessageType = "current_task_status"
	TypeAgentShutdown    MessageType = "agent_shutdown"

	// Server -> Agent messages
	TypeTaskAssignment   MessageType = "task_assignment"
	TypeJobStop          MessageType = "job_stop"
	TypeBenchmarkRequest MessageType = "benchmark_request"
	TypeAgentCommand     MessageType = "agent_command"
	TypeConfigUpdate     MessageType = "config_update"
	TypeSyncRequest      MessageType = "file_sync_request"
	TypeSyncCommand      MessageType = "file_sync_command"
	TypeForceCleanup     MessageType = "force_cleanup"
	TypeBufferAck        MessageType = "buffer_ack"
)

// Client represents a connected agent
type Client struct {
	LastSeen time.Time
}

// Message represents a WebSocket message
type Message struct {
	Type         MessageType      `json:"type"`
	Payload      json.RawMessage  `json:"payload"`
	HardwareInfo *models.Hardware `json:"hardware_info,omitempty"`
	OSInfo       json.RawMessage  `json:"os_info,omitempty"`
}


// HeartbeatPayload represents a heartbeat message from agent
type HeartbeatPayload struct {
	AgentID     int     `json:"agent_id"`
	LoadAverage float64 `json:"load_average"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}

// TaskStatusPayload represents task status update from agent
type TaskStatusPayload struct {
	AgentID   int       `json:"agent_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

// AgentStatusPayload represents agent status update
type AgentStatusPayload struct {
	AgentID     int                    `json:"agent_id"`
	Status      string                 `json:"status"`
	Version     string                 `json:"version"`
	LastError   string                 `json:"last_error,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Environment map[string]string      `json:"environment"`
	OSInfo      map[string]interface{} `json:"os_info,omitempty"`
}

// ErrorReportPayload represents detailed error report from agent
type ErrorReportPayload struct {
	AgentID    int       `json:"agent_id"`
	Error      string    `json:"error"`
	Stack      string    `json:"stack"`
	Context    any       `json:"context"`
	ReportedAt time.Time `json:"reported_at"`
}

// FileSyncRequestPayload represents a request for the agent to report its current files
type FileSyncRequestPayload struct {
	RequestID string   `json:"request_id"`
	FileTypes []string `json:"file_types"`         // "wordlist", "rule", "binary", "hashlist"
	Category  string   `json:"category,omitempty"` // Filter by category if needed
}

// FileInfo represents information about a file for synchronization
type FileInfo struct {
	Name      string `json:"name"`
	MD5Hash   string `json:"md5_hash"` // MD5 hash used for synchronization
	Size      int64  `json:"size"`
	FileType  string `json:"file_type"` // "wordlist", "rule", "binary", "hashlist"
	Category  string `json:"category,omitempty"`
	ID        int    `json:"id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// FileSyncResponsePayload represents the agent's response with its current files
type FileSyncResponsePayload struct {
	RequestID string     `json:"request_id"`
	AgentID   int        `json:"agent_id"`
	Files     []FileInfo `json:"files"`
}

// FileSyncCommandPayload represents a command to download specific files
type FileSyncCommandPayload struct {
	RequestID string     `json:"request_id"`
	Action    string     `json:"action"` // "download", "verify", etc.
	Files     []FileInfo `json:"files"`
}

// FileSyncStatusPayload represents a status update for file synchronization
type FileSyncStatusPayload struct {
	RequestID string           `json:"request_id"`
	AgentID   int              `json:"agent_id"`
	Status    string           `json:"status"`   // "in_progress", "completed", "failed"
	Progress  int              `json:"progress"` // 0-100 percentage
	Results   []FileSyncResult `json:"results,omitempty"`
}

// FileSyncResult represents the result of a file sync operation
type FileSyncResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	MD5Hash string `json:"md5_hash,omitempty"`
}

// TaskAssignmentPayload represents a job task assignment sent to an agent
type TaskAssignmentPayload struct {
	TaskID          string   `json:"task_id"`
	JobExecutionID  string   `json:"job_execution_id"`
	HashlistID      int64    `json:"hashlist_id"`
	HashlistPath    string   `json:"hashlist_path"`
	AttackMode      int      `json:"attack_mode"`
	HashType        int      `json:"hash_type"`
	KeyspaceStart   int64    `json:"keyspace_start"`
	KeyspaceEnd     int64    `json:"keyspace_end"`
	WordlistPaths   []string `json:"wordlist_paths"`
	RulePaths       []string `json:"rule_paths"`
	Mask            string   `json:"mask,omitempty"`
	BinaryPath      string   `json:"binary_path"`
	ChunkDuration   int      `json:"chunk_duration"`
	ReportInterval  int      `json:"report_interval"`
	OutputFormat    string   `json:"output_format"`
	ExtraParameters string   `json:"extra_parameters,omitempty"`
	EnabledDevices  []int    `json:"enabled_devices,omitempty"`
}

// BenchmarkResultPayload represents benchmark results from an agent
type BenchmarkResultPayload struct {
	AttackMode   int           `json:"attack_mode"`
	HashType     int           `json:"hash_type"`
	Speed        int64         `json:"speed"`                   // Total hashes per second
	DeviceSpeeds []DeviceSpeed `json:"device_speeds,omitempty"` // Per-device speeds
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
}

// DeviceSpeed represents speed for a single device
type DeviceSpeed struct {
	DeviceID   int    `json:"device_id"`
	DeviceName string `json:"device_name"`
	Speed      int64  `json:"speed"` // H/s for this device
}

// JobStopPayload represents a job stop command
type JobStopPayload struct {
	TaskID         string `json:"task_id"`
	JobExecutionID string `json:"job_execution_id"`
	Reason         string `json:"reason"`
}

// BenchmarkRequestPayload represents a benchmark request sent to an agent
type BenchmarkRequestPayload struct {
	RequestID  string `json:"request_id"`
	AttackMode int    `json:"attack_mode"`
	HashType   int    `json:"hash_type"`
	BinaryPath string `json:"binary_path"`
	// Additional fields for real-world speed test
	TaskID          string   `json:"task_id,omitempty"`
	HashlistID      int64    `json:"hashlist_id,omitempty"`
	HashlistPath    string   `json:"hashlist_path,omitempty"`
	WordlistPaths   []string `json:"wordlist_paths,omitempty"`
	RulePaths       []string `json:"rule_paths,omitempty"`
	Mask            string   `json:"mask,omitempty"`
	TestDuration    int      `json:"test_duration,omitempty"`    // Duration in seconds for speed test
	TimeoutDuration int      `json:"timeout_duration,omitempty"` // Maximum time to wait for speedtest (seconds)
	ExtraParameters string   `json:"extra_parameters,omitempty"` // Agent-specific hashcat parameters
	EnabledDevices  []int    `json:"enabled_devices,omitempty"`  // List of enabled device IDs
}

// Service handles WebSocket business logic
type Service struct {
	agentService *services.AgentService
	clients      map[int]*Client
	mu           sync.RWMutex
	jobHandler   JobHandler // Interface for handling job-related messages
}

// NewService creates a new WebSocket service
func NewService(agentService *services.AgentService) *Service {
	return &Service{
		agentService: agentService,
		clients:      make(map[int]*Client),
	}
}

// SetJobHandler sets the job handler for processing job-related messages
func (s *Service) SetJobHandler(handler JobHandler) {
	s.jobHandler = handler
}

// GetJobHandler returns the job handler for processing job-related messages
func (s *Service) GetJobHandler() JobHandler {
	return s.jobHandler
}

// HandleMessage processes incoming WebSocket messages
func (s *Service) HandleMessage(ctx context.Context, agent *models.Agent, msg *Message) error {
	// Update heartbeat on ANY message received from the agent
	// This ensures the agent is considered alive as long as it's communicating
	if err := s.agentService.UpdateHeartbeat(ctx, agent.ID); err != nil {
		// Log but don't fail the message processing
		fmt.Printf("Warning: failed to update heartbeat for agent %d: %v\n", agent.ID, err)
	}

	switch msg.Type {
	case TypeHeartbeat:
		return s.handleHeartbeat(ctx, agent, msg)
	case TypeTaskStatus:
		return s.handleTaskStatus(ctx, agent, msg)
	case TypeJobProgress:
		return s.handleJobProgress(ctx, agent, msg)
	case TypeBenchmarkResult:
		return s.handleBenchmarkResult(ctx, agent, msg)
	case TypeAgentStatus:
		return s.handleAgentStatus(ctx, agent, msg)
	case TypeErrorReport:
		return s.handleErrorReport(ctx, agent, msg)
	case TypeHardwareInfo:
		return s.handleHardwareInfo(ctx, agent, msg)
	case TypeSyncRequest:
		return s.handleSyncRequest(ctx, agent, msg)
	case TypeSyncCommand:
		return s.handleSyncCommand(ctx, agent, msg)
	case TypeHashcatOutput:
		return s.handleHashcatOutput(ctx, agent, msg)
	case TypeCurrentTaskStatus:
		// Current task status is handled in the handler layer
		// Just update heartbeat here
		return nil
	case TypeAgentShutdown:
		// Agent shutdown is handled in the handler layer
		// Just update heartbeat here
		return nil
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// updateLastSeen updates the last seen timestamp for an agent
func (s *Service) updateLastSeen(agentID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client, ok := s.clients[agentID]; ok {
		client.LastSeen = time.Now()
	}
}

// GetLastSeen returns when an agent was last seen
func (s *Service) GetLastSeen(agentID int) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if client, ok := s.clients[agentID]; ok {
		return client.LastSeen
	}
	return time.Time{}
}

// handleHeartbeat processes heartbeat messages
func (s *Service) handleHeartbeat(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload HeartbeatPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	// Update agent status in database
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, models.AgentStatusActive, nil); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	s.updateLastSeen(agent.ID)
	return nil
}


// handleTaskStatus processes task status messages
func (s *Service) handleTaskStatus(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload TaskStatusPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task status: %w", err)
	}

	// TODO: Update task status in task service
	return nil
}

// handleAgentStatus processes agent status messages
func (s *Service) handleAgentStatus(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload AgentStatusPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal agent status: %w", err)
	}

	// Update agent status in database
	var lastError *string
	if payload.LastError != "" {
		lastError = &payload.LastError
	}

	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, payload.Status, lastError); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	// Update agent version if provided
	if payload.Version != "" {
		if err := s.agentService.UpdateAgentVersion(ctx, agent.ID, payload.Version); err != nil {
			// Log error but don't fail the status update
			debug.Error("Failed to update agent version: %v", err)
		} else {
			debug.Info("Updated agent %d version to %s", agent.ID, payload.Version)
		}
	}

	// Update OS info if provided
	if payload.OSInfo != nil && len(payload.OSInfo) > 0 {
		if err := s.agentService.UpdateAgentOSInfo(ctx, agent.ID, payload.OSInfo); err != nil {
			// Log error but don't fail the status update
			debug.Error("Failed to update agent OS info: %v", err)
		}
	}

	return nil
}

// handleErrorReport processes error report messages
func (s *Service) handleErrorReport(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload ErrorReportPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal error report: %w", err)
	}

	// Update agent status with error
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "error", &payload.Error); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// handleHardwareInfo processes hardware information messages
func (s *Service) handleHardwareInfo(ctx context.Context, agent *models.Agent, msg *Message) error {
	// If HardwareInfo is not directly populated, try to unmarshal from Payload
	var hardware models.Hardware
	if err := json.Unmarshal(msg.Payload, &hardware); err != nil {
		return fmt.Errorf("failed to unmarshal hardware info: %w", err)
	}

	// Update agent's hardware information in the database
	agent.Hardware = hardware
	if err := s.agentService.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent hardware info: %w", err)
	}

	return nil
}

// handleSyncRequest processes file sync request messages
func (s *Service) handleSyncRequest(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload FileSyncRequestPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal file sync request: %w", err)
	}

	// Log the request
	fmt.Printf("Received file sync request from agent %d: %+v\n", agent.ID, payload)

	// This function should just acknowledge receipt of the request
	// The actual file comparison happens in the WebSocket handler

	// Update agent metadata to indicate sync is in progress
	if agent.Metadata == nil {
		agent.Metadata = make(map[string]string)
	}
	agent.Metadata["sync_request_id"] = payload.RequestID
	agent.Metadata["sync_status"] = "requested"
	agent.Metadata["sync_timestamp"] = fmt.Sprintf("%d", time.Now().Unix())

	if err := s.agentService.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent metadata for sync request: %w", err)
	}

	return nil
}

// handleSyncCommand processes file sync command messages
func (s *Service) handleSyncCommand(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload FileSyncCommandPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal file sync command: %w", err)
	}

	// Log the command
	fmt.Printf("Received file sync command for agent %d: action=%s, files=%d\n",
		agent.ID, payload.Action, len(payload.Files))

	// Update agent metadata to indicate sync command sent
	if agent.Metadata == nil {
		agent.Metadata = make(map[string]string)
	}
	agent.Metadata["sync_request_id"] = payload.RequestID
	agent.Metadata["sync_status"] = "command_received"
	agent.Metadata["sync_action"] = payload.Action
	agent.Metadata["sync_files_count"] = fmt.Sprintf("%d", len(payload.Files))
	agent.Metadata["sync_timestamp"] = fmt.Sprintf("%d", time.Now().Unix())

	if err := s.agentService.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent metadata for sync command: %w", err)
	}

	return nil
}

// handleJobProgress processes job progress messages from agents
func (s *Service) handleJobProgress(ctx context.Context, agent *models.Agent, msg *Message) error {
	// If no job handler is set, just log and ignore
	if s.jobHandler == nil {
		fmt.Printf("Received job progress from agent %d but no job handler set\n", agent.ID)
		return nil
	}

	// Process job progress asynchronously to avoid blocking the read loop
	go func() {
		// Create a new context with timeout for the async operation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.jobHandler.ProcessJobProgress(asyncCtx, agent.ID, msg.Payload); err != nil {
			debug.Error("Failed to process job progress from agent %d: %v", agent.ID, err)
		}
	}()

	return nil
}

// handleBenchmarkResult processes benchmark result messages from agents
func (s *Service) handleBenchmarkResult(ctx context.Context, agent *models.Agent, msg *Message) error {
	// If no job handler is set, just log and ignore
	if s.jobHandler == nil {
		fmt.Printf("Received benchmark result from agent %d but no job handler set\n", agent.ID)
		return nil
	}

	// Forward to job handler
	return s.jobHandler.ProcessBenchmarkResult(ctx, agent.ID, msg.Payload)
}

// handleHashcatOutput processes hashcat output messages from agents
func (s *Service) handleHashcatOutput(ctx context.Context, agent *models.Agent, msg *Message) error {
	// Process hashcat output asynchronously to avoid blocking the read loop
	go func() {
		var payload struct {
			TaskID    string    `json:"task_id"`
			Output    string    `json:"output"`
			IsError   bool      `json:"is_error"`
			Timestamp time.Time `json:"timestamp"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			debug.Error("Failed to unmarshal hashcat output from agent %d: %v", agent.ID, err)
			return
		}

		// Log the output for debugging
		if payload.IsError {
			fmt.Printf("[Agent %d][Task %s][ERROR] %s\n", agent.ID, payload.TaskID, payload.Output)
		} else {
			fmt.Printf("[Agent %d][Task %s] %s\n", agent.ID, payload.TaskID, payload.Output)
		}

		// TODO: Store output in database or forward to interested parties via SSE
	}()

	return nil
}

// HandleAgentDisconnection handles when an agent disconnects unexpectedly
func (s *Service) HandleAgentDisconnection(ctx context.Context, agentID int) error {
	// Check if we have a job handler
	if s.jobHandler == nil {
		debug.Warning("Agent %d disconnected but no job handler available to mark tasks", agentID)
		return nil
	}
	
	// Call the job handler to mark tasks as reconnect_pending
	// We use a type assertion to check if the handler supports disconnection handling
	type disconnectionHandler interface {
		HandleAgentDisconnection(ctx context.Context, agentID int) error
	}
	
	if handler, ok := s.jobHandler.(disconnectionHandler); ok {
		return handler.HandleAgentDisconnection(ctx, agentID)
	}
	
	debug.Warning("Job handler does not support disconnection handling")
	return nil
}
