package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Agent status constants
const (
	AgentStatusPending  = "pending"
	AgentStatusActive   = "active"
	AgentStatusInactive = "inactive"
	AgentStatusError    = "error"
	AgentStatusDisabled = "disabled"
)

// Agent represents a registered agent in the system
type Agent struct {
	ID                  int               `json:"id"`
	Name                string            `json:"name"`
	Status              string            `json:"status"`
	LastError           sql.NullString    `json:"lastError"`
	LastSeen            time.Time         `json:"lastSeen"`
	LastHeartbeat       time.Time         `json:"lastHeartbeat"`
	Version             string            `json:"version"`
	Hardware            Hardware          `json:"hardware"`
	OSInfo              json.RawMessage   `json:"os_info"`
	CreatedByID         uuid.UUID         `json:"createdById"`
	CreatedBy           *User             `json:"createdBy,omitempty"`
	Teams               []Team            `json:"teams,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
	APIKey              sql.NullString    `json:"-"`
	APIKeyCreatedAt     sql.NullTime      `json:"-"`
	APIKeyLastUsed      sql.NullTime      `json:"-"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	OwnerID             *uuid.UUID        `json:"ownerId,omitempty"`
	ExtraParameters     string            `json:"extraParameters"`
	IsEnabled           bool              `json:"isEnabled"`
	ConsecutiveFailures int               `json:"consecutiveFailures"` // Track consecutive task failures
}

// Hardware represents the hardware configuration of an agent
type Hardware struct {
	CPUs              []CPU              `json:"cpus"`
	GPUs              []GPU              `json:"gpus"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
}

// CPU represents a CPU in the agent's hardware
type CPU struct {
	Model       string  `json:"model"`
	Cores       int     `json:"cores"`
	Threads     int     `json:"threads"`
	Frequency   float64 `json:"frequency"`
	Temperature float64 `json:"temperature"`
}

// GPU represents a GPU in the agent's hardware
type GPU struct {
	Vendor      string  `json:"vendor"`
	Model       string  `json:"model"`
	Memory      int64   `json:"memory"`
	Driver      string  `json:"driver"`
	Temperature float64 `json:"temperature"`
	PowerUsage  float64 `json:"powerUsage"`
	Utilization float64 `json:"utilization"`
}

// NetworkInterface represents a network interface in the agent's hardware
type NetworkInterface struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
}

// AgentMetrics represents metrics collected from an agent
type AgentMetrics struct {
	ID             int             `json:"id"`
	AgentID        int             `json:"agent_id"`
	CPUUsage       float64         `json:"cpu_usage"`
	MemoryUsage    float64         `json:"memory_usage"`
	GPUUtilization float64         `json:"gpu_utilization"`
	GPUTemp        float64         `json:"gpu_temp"`
	GPUMetrics     json.RawMessage `json:"gpu_metrics"`
	Timestamp      time.Time       `json:"timestamp"`
}

// ScanHardware scans a JSON-encoded hardware string into the Hardware struct
func (a *Agent) ScanHardware(value interface{}) error {
	if value == nil {
		a.Hardware = Hardware{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &a.Hardware)
	case string:
		return json.Unmarshal([]byte(v), &a.Hardware)
	default:
		return fmt.Errorf("unsupported type for hardware: %T", value)
	}
}

// Value returns the JSON encoding of Hardware for database storage
func (h Hardware) Value() (driver.Value, error) {
	return json.Marshal(h)
}
