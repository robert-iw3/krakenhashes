package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/mitchellh/mapstructure"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Agent -> Server messages
	TypeHeartbeat   MessageType = "heartbeat"
	TypeMetrics     MessageType = "metrics"
	TypeTaskStatus  MessageType = "task_status"
	TypeAgentStatus MessageType = "agent_status"
	TypeErrorReport MessageType = "error_report"

	// Server -> Agent messages
	TypeTaskAssignment MessageType = "task_assignment"
	TypeAgentCommand   MessageType = "agent_command"
	TypeConfigUpdate   MessageType = "config_update"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType    `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// MetricsPayload represents detailed metrics from agent
type MetricsPayload struct {
	AgentID       string    `json:"agent_id"`
	CollectedAt   time.Time `json:"collected_at"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskUsage     float64   `json:"disk_usage"`
	NetworkStats  any       `json:"network_stats"`
	ProcessStats  any       `json:"process_stats"`
	CustomMetrics any       `json:"custom_metrics,omitempty"`
}

// HeartbeatPayload represents a heartbeat message from agent
type HeartbeatPayload struct {
	AgentID     string  `json:"agent_id"`
	LoadAverage float64 `json:"load_average"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}

// TaskStatusPayload represents task status update from agent
type TaskStatusPayload struct {
	AgentID   string    `json:"agent_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

// AgentStatusPayload represents agent status update
type AgentStatusPayload struct {
	AgentID     string            `json:"agent_id"`
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	LastError   string            `json:"last_error,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Environment map[string]string `json:"environment"`
}

// ErrorReportPayload represents detailed error report from agent
type ErrorReportPayload struct {
	AgentID    string    `json:"agent_id"`
	Error      string    `json:"error"`
	Stack      string    `json:"stack"`
	Context    any       `json:"context"`
	ReportedAt time.Time `json:"reported_at"`
}

// Service handles WebSocket business logic
type Service struct {
	agentService *services.AgentService
	metrics      map[string]*MetricsPayload
	metricsMu    sync.RWMutex
	lastSeen     map[string]time.Time
	lastSeenMu   sync.RWMutex
}

// NewService creates a new WebSocket service
func NewService(agentService *services.AgentService) *Service {
	return &Service{
		agentService: agentService,
		metrics:      make(map[string]*MetricsPayload),
		lastSeen:     make(map[string]time.Time),
	}
}

// HandleMessage processes incoming WebSocket messages
func (s *Service) HandleMessage(ctx context.Context, agent *models.Agent, msg *Message) error {
	// Update last seen timestamp
	s.updateLastSeen(agent.ID)

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
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// updateLastSeen updates the last seen timestamp for an agent
func (s *Service) updateLastSeen(agentID string) {
	s.lastSeenMu.Lock()
	s.lastSeen[agentID] = time.Now()
	s.lastSeenMu.Unlock()
}

// GetLastSeen returns when an agent was last seen
func (s *Service) GetLastSeen(agentID string) time.Time {
	s.lastSeenMu.RLock()
	defer s.lastSeenMu.RUnlock()
	return s.lastSeen[agentID]
}

// handleHeartbeat processes heartbeat messages
func (s *Service) handleHeartbeat(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload HeartbeatPayload
	if err := mapstructure.Decode(msg.Payload, &payload); err != nil {
		return err
	}

	// Update agent status in database
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "online", nil); err != nil {
		return err
	}

	return nil
}

// handleMetrics processes metrics messages
func (s *Service) handleMetrics(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload MetricsPayload
	if err := mapstructure.Decode(msg.Payload, &payload); err != nil {
		return err
	}

	// Store latest metrics
	s.metricsMu.Lock()
	s.metrics[agent.ID] = &payload
	s.metricsMu.Unlock()

	return nil
}

// handleTaskStatus processes task status messages
func (s *Service) handleTaskStatus(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload TaskStatusPayload
	if err := mapstructure.Decode(msg.Payload, &payload); err != nil {
		return err
	}

	// TODO: Update task status in task service
	return nil
}

// handleAgentStatus processes agent status messages
func (s *Service) handleAgentStatus(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload AgentStatusPayload
	if err := mapstructure.Decode(msg.Payload, &payload); err != nil {
		return err
	}

	// Update agent status in database
	var lastError *string
	if payload.LastError != "" {
		lastError = &payload.LastError
	}

	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, payload.Status, lastError); err != nil {
		return err
	}

	return nil
}

// handleErrorReport processes error report messages
func (s *Service) handleErrorReport(ctx context.Context, agent *models.Agent, msg *Message) error {
	var payload ErrorReportPayload
	if err := mapstructure.Decode(msg.Payload, &payload); err != nil {
		return err
	}

	// Update agent status with error
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "error", &payload.Error); err != nil {
		return err
	}

	return nil
}
