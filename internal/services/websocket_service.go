package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/handlers/websocket"
	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// WebSocketService handles WebSocket business logic
type WebSocketService struct {
	agentService *AgentService
	metrics      map[uint]*websocket.MetricsPayload
	metricsMu    sync.RWMutex
	lastSeen     map[uint]time.Time
	lastSeenMu   sync.RWMutex
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(agentService *AgentService) *WebSocketService {
	return &WebSocketService{
		agentService: agentService,
		metrics:      make(map[uint]*websocket.MetricsPayload),
		lastSeen:     make(map[uint]time.Time),
	}
}

// HandleMessage processes incoming WebSocket messages
func (s *WebSocketService) HandleMessage(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	// Update last seen timestamp
	s.updateLastSeen(agent.ID)

	switch msg.Type {
	case websocket.TypeHeartbeat:
		return s.handleHeartbeat(ctx, agent, msg)
	case websocket.TypeMetrics:
		return s.handleMetrics(ctx, agent, msg)
	case websocket.TypeTaskStatus:
		return s.handleTaskStatus(ctx, agent, msg)
	case websocket.TypeAgentStatus:
		return s.handleAgentStatus(ctx, agent, msg)
	case websocket.TypeErrorReport:
		return s.handleErrorReport(ctx, agent, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleHeartbeat processes heartbeat messages
func (s *WebSocketService) handleHeartbeat(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	var payload websocket.HeartbeatPayload
	if err := s.unmarshalPayload(msg.Payload, &payload); err != nil {
		return err
	}

	// Update agent status in database
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "online", nil); err != nil {
		debug.Error("failed to update agent status: %v", err)
		return err
	}

	return nil
}

// handleMetrics processes metrics messages
func (s *WebSocketService) handleMetrics(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	var payload websocket.MetricsPayload
	if err := s.unmarshalPayload(msg.Payload, &payload); err != nil {
		return err
	}

	// Store latest metrics
	s.metricsMu.Lock()
	s.metrics[agent.ID] = &payload
	s.metricsMu.Unlock()

	return nil
}

// handleTaskStatus processes task status messages
func (s *WebSocketService) handleTaskStatus(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	var payload websocket.TaskStatusPayload
	if err := s.unmarshalPayload(msg.Payload, &payload); err != nil {
		return err
	}

	// TODO: Update task status in task service
	return nil
}

// handleAgentStatus processes agent status messages
func (s *WebSocketService) handleAgentStatus(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	var payload websocket.AgentStatusPayload
	if err := s.unmarshalPayload(msg.Payload, &payload); err != nil {
		return err
	}

	// Update agent status in database
	var lastError *string
	if payload.LastError != "" {
		lastError = &payload.LastError
	}

	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, payload.Status, lastError); err != nil {
		debug.Error("failed to update agent status: %v", err)
		return err
	}

	return nil
}

// handleErrorReport processes error report messages
func (s *WebSocketService) handleErrorReport(ctx context.Context, agent *models.Agent, msg *websocket.Message) error {
	var payload websocket.ErrorReportPayload
	if err := s.unmarshalPayload(msg.Payload, &payload); err != nil {
		return err
	}

	// Log error report
	debug.Error("agent %d error report: %s\nStack: %s\nContext: %v",
		agent.ID, payload.Error, payload.Stack, payload.Context)

	// Update agent status with error
	if err := s.agentService.UpdateAgentStatus(ctx, agent.ID, "error", &payload.Error); err != nil {
		debug.Error("failed to update agent status: %v", err)
		return err
	}

	return nil
}

// GetAgentMetrics returns the latest metrics for an agent
func (s *WebSocketService) GetAgentMetrics(agentID uint) *websocket.MetricsPayload {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	return s.metrics[agentID]
}

// GetLastSeen returns when an agent was last seen
func (s *WebSocketService) GetLastSeen(agentID uint) time.Time {
	s.lastSeenMu.RLock()
	defer s.lastSeenMu.RUnlock()
	return s.lastSeen[agentID]
}

// updateLastSeen updates the last seen timestamp for an agent
func (s *WebSocketService) updateLastSeen(agentID uint) {
	s.lastSeenMu.Lock()
	s.lastSeen[agentID] = time.Now()
	s.lastSeenMu.Unlock()
}

// unmarshalPayload helper to unmarshal message payloads
func (s *WebSocketService) unmarshalPayload(payload map[string]interface{}, v interface{}) error {
	return websocket.UnmarshalPayload(payload, v)
}
