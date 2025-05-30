package models

import "time"

// SystemSetting represents a key-value pair for global system settings.
type SystemSetting struct {
	Key         string    `json:"key" db:"key"`
	Value       *string   `json:"value,omitempty" db:"value"` // Use pointer to handle NULL-able TEXT
	Description *string   `json:"description,omitempty" db:"description"`
	DataType    string    `json:"data_type" db:"data_type"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// MaxPriorityConfig represents the max priority configuration
type MaxPriorityConfig struct {
	MaxPriority int `json:"max_priority"`
}
