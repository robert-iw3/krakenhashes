package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom/backend/internal/models"
	"github.com/ZerkerEOD/hashdom/backend/internal/services"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Agent -> Server messages
	TypeHeartbeat    MessageType = "heartbeat"
	TypeMetrics      MessageType = "metrics"
	TypeTaskStatus   MessageType = "task_status"
	TypeAgentStatus  MessageType = "agent_status"
	TypeErrorReport  MessageType = "error_report"
	TypeHardwareInfo MessageType = "hardware_info"

	// Server -> Agent messages
	TypeTaskAssignment MessageType = "task_assignment"
	TypeAgentCommand   MessageType = "agent_command"
	TypeConfigUpdate   MessageType = "config_update"
)

// Client represents a connected agent
type Client struct {
	LastSeen time.Time
	Metrics  *MetricsPayload
}

// Message represents a WebSocket message
type Message struct {
	Type         MessageType      `json:"type"`
	Payload      json.RawMessage  `json:"payload"`
	HardwareInfo *models.Hardware `json:"hardware_info,omitempty"`
	OSInfo       json.RawMessage  `json:"os_info,omitempty"`
}

// MetricsPayload represents detailed metrics from agent
type MetricsPayload struct {
	AgentID        int       `json:"agent_id"`
	CollectedAt    time.Time `json:"collected_at"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	DiskUsage      float64   `json:"disk_usage"`
	GPUUtilization float64   `json:"gpu_utilization"`
	GPUTemp        float64   `json:"gpu_temp"`
	NetworkStats   any       `json:"network_stats"`
	ProcessStats   any       `json:"process_stats"`
	CustomMetrics  any       `json:"custom_metrics,omitempty"`
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
	AgentID     int               `json:"agent_id"`
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	LastError   string            `json:"last_error,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Environment map[string]string `json:"environment"`
}

// ErrorReportPayload represents detailed error report from agent
type ErrorReportPayload struct {
	AgentID    int       `json:"agent_id"`
	Error      string    `json:"error"`
	Stack      string    `json:"stack"`
	Context    any       `json:"context"`
	ReportedAt time.Time `json:"reported_at"`
}

// Service handles WebSocket business logic
type Service struct {
	agentService *services.AgentService
	clients      map[int]*Client
	mu           sync.RWMutex
}

// NewService creates a new WebSocket service
func NewService(agentService *services.AgentService) *Service {
	return &Service{
		agentService: agentService,
		clients:      make(map[int]*Client),
	}
}

// HandleMessage processes incoming WebSocket messages
func (s *Service) HandleMessage(ctx context.Context, agent *models.Agent, msg *Message) error {
	switch msg.Type {
	case TypeHeartbeat:
		return s.handleHeartbeat(ctx, agent, msg)
	case TypeMetrics:
		return s.handleMetrics(ctx, agent, msg)
	case TypeTaskStatus:
		return s.handleTaskStatus(ctx, agent, msg)
	case TypeAgentStatus:
		return s.handleAgentStatus(ctx, agent, msg)
	case TypeErrorReport:
		return s.handleErrorReport(ctx, agent, msg)
	case TypeHardwareInfo:
		return s.handleHardwareInfo(ctx, agent, msg)
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
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "online", nil); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	s.updateLastSeen(agent.ID)
	return nil
}

// handleMetrics processes metrics messages
func (s *Service) handleMetrics(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload MetricsPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	// Store latest metrics with the client
	s.mu.Lock()
	if client, ok := s.clients[agent.ID]; ok {
		client.Metrics = &payload
	}
	s.mu.Unlock()

	// Convert and store metrics in database
	metrics := &models.AgentMetrics{
		AgentID:        agent.ID,
		CPUUsage:       payload.CPUUsage,
		MemoryUsage:    payload.MemoryUsage,
		GPUUtilization: payload.GPUUtilization,
		GPUTemp:        payload.GPUTemp,
		GPUMetrics:     msg.Payload, // Store additional GPU metrics as JSON
		Timestamp:      payload.CollectedAt,
	}

	if err := s.agentService.ProcessMetrics(ctx, agent.ID, metrics); err != nil {
		return fmt.Errorf("failed to process metrics: %w", err)
	}

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
