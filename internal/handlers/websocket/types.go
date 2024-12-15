package websocket

import (
	"encoding/json"
	"time"
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

// HeartbeatPayload represents a heartbeat message from agent
type HeartbeatPayload struct {
	AgentID     uint    `json:"agent_id"`
	LoadAverage float64 `json:"load_average"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}

// MetricsPayload represents detailed metrics from agent
type MetricsPayload struct {
	AgentID       uint      `json:"agent_id"`
	CollectedAt   time.Time `json:"collected_at"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskUsage     float64   `json:"disk_usage"`
	NetworkStats  any       `json:"network_stats"`
	ProcessStats  any       `json:"process_stats"`
	CustomMetrics any       `json:"custom_metrics,omitempty"`
}

// TaskStatusPayload represents task status update from agent
type TaskStatusPayload struct {
	AgentID   uint      `json:"agent_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

// AgentStatusPayload represents agent status update
type AgentStatusPayload struct {
	AgentID     uint              `json:"agent_id"`
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	LastError   string            `json:"last_error,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Environment map[string]string `json:"environment"`
}

// ErrorReportPayload represents detailed error report from agent
type ErrorReportPayload struct {
	AgentID    uint      `json:"agent_id"`
	Error      string    `json:"error"`
	Stack      string    `json:"stack"`
	Context    any       `json:"context"`
	ReportedAt time.Time `json:"reported_at"`
}

// TaskAssignmentPayload represents a task assignment to agent
type TaskAssignmentPayload struct {
	TaskID     string         `json:"task_id"`
	Type       string         `json:"type"`
	Priority   int            `json:"priority"`
	Parameters map[string]any `json:"parameters"`
	Deadline   *time.Time     `json:"deadline,omitempty"`
}

// AgentCommandPayload represents a command to be executed by agent
type AgentCommandPayload struct {
	Command    string         `json:"command"`
	Parameters map[string]any `json:"parameters"`
	Timeout    time.Duration  `json:"timeout,omitempty"`
}

// ConfigUpdatePayload represents configuration update for agent
type ConfigUpdatePayload struct {
	Settings map[string]any `json:"settings"`
	Version  int            `json:"version"`
}

// ToMessage converts a payload to a Message
func ToMessage(msgType MessageType, payload interface{}) (*Message, error) {
	payloadMap := make(map[string]any)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
		return nil, err
	}

	return &Message{
		Type:      msgType,
		Timestamp: time.Now(),
		Payload:   payloadMap,
	}, nil
}
