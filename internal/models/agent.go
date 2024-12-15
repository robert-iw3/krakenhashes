package models

import (
	"time"
)

// Hardware represents the hardware information of an agent
type Hardware struct {
	CPUs              []CPU              `json:"cpus"`
	GPUs              []GPU              `json:"gpus"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

// CPU represents CPU information
type CPU struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
}

// GPU represents GPU information
type GPU struct {
	Model  string `json:"model"`
	Memory string `json:"memory"`
	Driver string `json:"driver"`
}

// NetworkInterface represents network interface information
type NetworkInterface struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
}

// Agent represents a registered agent in the system
type Agent struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"not null"`
	Status        string    `json:"status" gorm:"not null;default:'inactive'"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	Version       string    `json:"version" gorm:"not null"`
	Hardware      Hardware  `json:"hardware" gorm:"type:jsonb"`
	CreatedByID   uint      `json:"createdById" gorm:"not null"`
	CreatedBy     User      `json:"createdBy" gorm:"foreignKey:CreatedByID"`
	Teams         []Team    `json:"teams" gorm:"many2many:agent_teams"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Certificate   string    `json:"-" gorm:"type:text"` // PEM encoded certificate
	PrivateKey    string    `json:"-" gorm:"type:text"` // PEM encoded private key (stored temporarily)
}

// AgentMetrics represents real-time metrics from an agent
type AgentMetrics struct {
	AgentID     uint      `json:"agentId" gorm:"primaryKey"`
	CPUUsage    float64   `json:"cpuUsage"`
	GPUUsage    float64   `json:"gpuUsage"`
	GPUTemp     float64   `json:"gpuTemp"`
	MemoryUsage float64   `json:"memoryUsage"`
	Timestamp   time.Time `json:"timestamp" gorm:"primaryKey"`
}

// TableName specifies the table name for AgentMetrics
func (AgentMetrics) TableName() string {
	return "agent_metrics"
}

// ValidStatus returns whether the given status is valid
func ValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"inactive": true,
		"active":   true,
		"error":    true,
	}
	return validStatuses[status]
}
