package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AttackMode represents the Hashcat attack mode.
// Valid values: 0, 1, 3, 6, 7, 9.
type AttackMode int

const (
	AttackModeStraight           AttackMode = 0 // Straight
	AttackModeCombination        AttackMode = 1 // Combination
	AttackModeBruteForce         AttackMode = 3 // Brute-force
	AttackModeHybridWordlistMask AttackMode = 6 // Hybrid Wordlist + Mask
	AttackModeHybridMaskWordlist AttackMode = 7 // Hybrid Mask + Wordlist
	AttackModeAssociation        AttackMode = 9 // Association
)

// IDArray is a custom type for handling arrays of IDs stored as JSONB in PostgreSQL
type IDArray []string

// Value implements the driver.Valuer interface
func (a IDArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *IDArray) Scan(value interface{}) error {
	if value == nil {
		*a = IDArray{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for IDArray: %T", value)
	}

	return json.Unmarshal(bytes, a)
}

// PresetJob mirrors the preset_jobs table structure.
// It defines a pre-configured set of parameters for a cracking job.
type PresetJob struct {
	ID                        uuid.UUID  `json:"id" db:"id"`
	Name                      string     `json:"name" db:"name"`
	WordlistIDs               IDArray    `json:"wordlist_ids" db:"wordlist_ids"` // Stores numeric IDs as strings in JSONB
	RuleIDs                   IDArray    `json:"rule_ids" db:"rule_ids"`         // Stores numeric IDs as strings in JSONB
	AttackMode                AttackMode `json:"attack_mode" db:"attack_mode"`
	Priority                  int        `json:"priority" db:"priority"`
	ChunkSizeSeconds          int        `json:"chunk_size_seconds" db:"chunk_size_seconds"`
	StatusUpdatesEnabled      bool       `json:"status_updates_enabled" db:"status_updates_enabled"`
	IsSmallJob                bool       `json:"is_small_job" db:"is_small_job"`
	AllowHighPriorityOverride bool       `json:"allow_high_priority_override" db:"allow_high_priority_override"`
	BinaryVersionID           int        `json:"binary_version_id" db:"binary_version_id"` // References binary_versions.id
	Mask                      string     `json:"mask,omitempty" db:"mask"`                 // For mask-based attack modes
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`

	// Fields potentially populated by JOINs in specific queries
	BinaryVersionName string `json:"binary_version_name,omitempty" db:"binary_version_name"` // Example: Populated when listing
}

// JobWorkflow mirrors the job_workflows table structure.
// It represents a named sequence of preset jobs.
type JobWorkflow struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Populated field holding the ordered steps
	Steps []JobWorkflowStep `json:"steps,omitempty"`
}

// JobWorkflowStep mirrors the job_workflow_steps table structure.
// It links a JobWorkflow to a PresetJob at a specific order.
type JobWorkflowStep struct {
	ID            int64     `json:"id" db:"id"` // Using int64 for BIGSERIAL
	JobWorkflowID uuid.UUID `json:"job_workflow_id" db:"job_workflow_id"`
	PresetJobID   uuid.UUID `json:"preset_job_id" db:"preset_job_id"`
	StepOrder     int       `json:"step_order" db:"step_order"`

	// Fields potentially populated by JOINs
	PresetJobName string `json:"preset_job_name,omitempty" db:"preset_job_name"`
}

// PresetJobBasic represents minimal information about a preset job
// Used for dropdowns and selection interfaces
type PresetJobBasic struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}
