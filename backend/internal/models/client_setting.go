package models

import "time"

// ClientSetting represents a key-value pair for global client settings.
type ClientSetting struct {
	Key         string    `json:"key"`
	Value       *string   `json:"value,omitempty"` // Use pointer to handle NULL-able TEXT
	Description *string   `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
