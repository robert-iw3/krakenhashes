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
)

// CertificateInfo stores information about an agent's certificate
type CertificateInfo struct {
	SerialNumber string    `json:"serial_number"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Agent represents a registered agent in the system
type Agent struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	LastError       sql.NullString  `json:"lastError"`
	LastSeen        time.Time       `json:"lastSeen"`
	LastHeartbeat   time.Time       `json:"lastHeartbeat"`
	Version         string          `json:"version"`
	Hardware        Hardware        `json:"hardware"`
	CreatedByID     uuid.UUID       `json:"createdById"`
	CreatedBy       *User           `json:"createdBy,omitempty"`
	Teams           []Team          `json:"teams,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	Certificate     sql.NullString  `json:"-"`
	PrivateKey      sql.NullString  `json:"-"`
	CertificateInfo CertificateInfo `json:"certificate_info"`
}

// Hardware represents the hardware configuration of an agent
type Hardware struct {
	CPUs              []CPU              `json:"cpus"`
	GPUs              []GPU              `json:"gpus"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
}

// CPU represents a CPU in the agent's hardware
type CPU struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
}

// GPU represents a GPU in the agent's hardware
type GPU struct {
	Model  string `json:"model"`
	Memory string `json:"memory"`
	Driver string `json:"driver"`
}

// NetworkInterface represents a network interface in the agent's hardware
type NetworkInterface struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
}

// AgentMetrics represents real-time metrics from an agent
type AgentMetrics struct {
	AgentID     string    `json:"agent_id"`
	CPUUsage    float64   `json:"cpu_usage"`
	GPUUsage    float64   `json:"gpu_usage"`
	GPUTemp     float64   `json:"gpu_temp"`
	MemoryUsage float64   `json:"memory_usage"`
	Timestamp   time.Time `json:"timestamp"`
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

// Scan implements the sql.Scanner interface for CertificateInfo
func (ci *CertificateInfo) Scan(value interface{}) error {
	if value == nil {
		*ci = CertificateInfo{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, ci)
	case string:
		return json.Unmarshal([]byte(v), ci)
	default:
		return fmt.Errorf("unsupported type for certificate_info: %T", value)
	}
}

// Value implements the driver.Valuer interface for CertificateInfo
func (ci CertificateInfo) Value() (driver.Value, error) {
	return json.Marshal(ci)
}
