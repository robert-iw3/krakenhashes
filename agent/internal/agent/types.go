/*
 * Package agent provides WebSocket communication types and structures
 * for the KrakenHashes agent.
 *
 * Message Types:
 *   - Job assignments
 *   - Status updates
 *   - Heartbeat messages
 *   - Error notifications
 *
 * Protocol:
 *   - All messages are JSON encoded
 *   - Bidirectional communication
 *   - Secure WebSocket (WSS) required
 *
 * Error Handling:
 *   - Message validation
 *   - Protocol violations
 *   - Connection failures
 */
package agent

import "time"

/*
 * MessageType defines the types of messages that can be exchanged
 * between the agent and backend.
 *
 * Values:
 *   - TypeJobAssignment: New job assignment
 *   - TypeStatusUpdate: Agent status report
 *   - TypeHeartbeat: Connection health check
 *   - TypeError: Error notification
 */
type MessageType string

const (
	TypeJobAssignment MessageType = "job_assignment"
	TypeStatusUpdate  MessageType = "status_update"
	TypeHeartbeat     MessageType = "heartbeat"
	TypeError         MessageType = "error"
)

/*
 * JobAssignment represents a job assignment message from the backend.
 *
 * Fields:
 *   - Type: Must be TypeJobAssignment
 *   - JobID: Unique identifier for the job
 *   - Parameters: Job-specific parameters
 *   - Priority: Job processing priority (1-10)
 *
 * Validation:
 *   - JobID must be non-empty
 *   - Priority must be within range
 *   - Parameters must be valid JSON
 */
type JobAssignment struct {
	Type       MessageType `json:"type"`
	JobID      string      `json:"job_id"`
	Parameters any         `json:"parameters"`
	Priority   int         `json:"priority"`
}

/*
 * StatusUpdate represents an agent status update message.
 *
 * Fields:
 *   - Type: Must be TypeStatusUpdate
 *   - JobID: Associated job identifier
 *   - Status: Current job status
 *   - Progress: Completion percentage
 *   - Details: Additional status information
 *
 * Status Values:
 *   - "initializing"
 *   - "processing"
 *   - "completed"
 *   - "failed"
 */
type StatusUpdate struct {
	Type     MessageType `json:"type"`
	JobID    string      `json:"job_id"`
	Status   string      `json:"status"`
	Progress float64     `json:"progress"`
	Details  any         `json:"details"`
}

/*
 * Heartbeat represents a bidirectional heartbeat message
 *
 * Fields:
 *   - Type: Must be TypeHeartbeat
 *   - Timestamp: Time of the heartbeat message
 *   - Status: Current connection status
 *   - Metrics: System performance metrics
 *
 * Validation:
 *   - Timestamp must be valid
 *   - Status must be a valid connection status
 *   - Metrics must be valid JSON
 */
type Heartbeat struct {
	Type      MessageType    `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Status    string         `json:"status"`
	Metrics   *SystemMetrics `json:"metrics,omitempty"`
}

/*
 * SystemMetrics represents system performance metrics
 *
 * Fields:
 *   - CPUUsage: Current CPU usage percentage
 *   - GPUUsage: Current GPU usage percentage
 *   - GPUTemp: Current GPU temperature
 *   - MemoryUsage: Current memory usage percentage
 *
 * Validation:
 *   - CPUUsage must be within range
 *   - GPUUsage must be within range
 *   - GPUTemp must be within range
 *   - MemoryUsage must be within range
 */
type SystemMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	GPUUtilization float64 `json:"gpu_utilization"`
	GPUTemp        float64 `json:"gpu_temp"`
}

type Message struct {
	Type    MessageType    `json:"type"`
	Metrics *SystemMetrics `json:"metrics,omitempty"`
}
