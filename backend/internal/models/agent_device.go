package models

import (
	"time"
)

// AgentDevice represents a compute device on an agent
type AgentDevice struct {
	ID         int       `json:"id" db:"id"`
	AgentID    int       `json:"agent_id" db:"agent_id"`
	DeviceID   int       `json:"device_id" db:"device_id"`
	DeviceName string    `json:"device_name" db:"device_name"`
	DeviceType string    `json:"device_type" db:"device_type"` // "GPU" or "CPU"
	Enabled    bool      `json:"enabled" db:"enabled"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// DeviceDetectionResult represents the result from agent device detection
type DeviceDetectionResult struct {
	Devices []Device `json:"devices"`
	Error   string   `json:"error,omitempty"`
}

// Device represents a compute device detected by hashcat
type Device struct {
	ID          int    `json:"device_id"`
	Name        string `json:"device_name"`
	Type        string `json:"device_type"` // "GPU" or "CPU"
	Enabled     bool   `json:"enabled"`
	
	// Additional properties from hashcat output
	Processors  int    `json:"processors,omitempty"`
	Clock       int    `json:"clock,omitempty"`       // MHz
	MemoryTotal int64  `json:"memory_total,omitempty"` // MB
	MemoryFree  int64  `json:"memory_free,omitempty"`  // MB
	PCIAddress  string `json:"pci_address,omitempty"`
	
	// Backend information
	Backend     string `json:"backend,omitempty"`      // "HIP", "OpenCL", "CUDA", etc.
	IsAlias     bool   `json:"is_alias,omitempty"`
	AliasOf     int    `json:"alias_of,omitempty"`     // Device ID this is an alias of
}

// DeviceUpdate represents a device update request
type DeviceUpdate struct {
	DeviceID int  `json:"device_id"`
	Enabled  bool `json:"enabled"`
}

// AgentWithDevices represents an agent with its devices
type AgentWithDevices struct {
	Agent
	Devices []AgentDevice `json:"devices"`
}