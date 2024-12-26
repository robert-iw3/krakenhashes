package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/db/queries"
	"github.com/ZerkerEOD/hashdom-backend/internal/models"
)

// AgentRepository handles database operations for agents
type AgentRepository struct {
	db *db.DB
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(db *db.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create creates a new agent
func (r *AgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	// Convert hardware to JSON
	hardwareJSON, err := json.Marshal(agent.Hardware)
	if err != nil {
		return fmt.Errorf("failed to marshal hardware: %w", err)
	}

	// Convert certificate info to JSON
	certInfoJSON, err := json.Marshal(agent.CertificateInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate info: %w", err)
	}

	err = r.db.QueryRowContext(ctx, queries.CreateAgent,
		agent.ID,
		agent.Name,
		agent.Status,
		agent.LastError,
		agent.LastSeen,
		agent.LastHeartbeat,
		agent.Version,
		hardwareJSON,
		agent.CreatedByID,
		agent.CreatedAt,
		agent.UpdatedAt,
		agent.Certificate,
		agent.PrivateKey,
		certInfoJSON,
	).Scan(&agent.ID)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// GetByID retrieves an agent by ID
func (r *AgentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	agent := &models.Agent{}
	var hardwareJSON, certInfoJSON []byte
	var createdByUser models.User

	err := r.db.QueryRowContext(ctx, queries.GetAgentByID, id).Scan(
		&agent.ID,
		&agent.Name,
		&agent.Status,
		&agent.LastError,
		&agent.LastSeen,
		&agent.LastHeartbeat,
		&agent.Version,
		&hardwareJSON,
		&agent.CreatedByID,
		&agent.CreatedAt,
		&agent.UpdatedAt,
		&agent.Certificate,
		&agent.PrivateKey,
		&certInfoJSON,
		&createdByUser.ID,
		&createdByUser.Username,
		&createdByUser.Email,
		&createdByUser.Role,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", id)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Unmarshal hardware JSON
	if err := json.Unmarshal(hardwareJSON, &agent.Hardware); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hardware: %w", err)
	}

	// Unmarshal certificate info JSON
	if err := json.Unmarshal(certInfoJSON, &agent.CertificateInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate info: %w", err)
	}

	agent.CreatedBy = &createdByUser
	return agent, nil
}

// ExistsByName checks if an agent exists with the given name
func (r *AgentRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM agents WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check agent existence: %w", err)
	}
	return exists, nil
}

// Update updates an agent
func (r *AgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	// Convert hardware to JSON
	hardwareJSON, err := json.Marshal(agent.Hardware)
	if err != nil {
		return fmt.Errorf("failed to marshal hardware: %w", err)
	}

	// Convert certificate info to JSON
	certInfoJSON, err := json.Marshal(agent.CertificateInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate info: %w", err)
	}

	result, err := r.db.ExecContext(ctx, queries.UpdateAgent,
		agent.ID,
		agent.Name,
		agent.Status,
		agent.LastError,
		agent.LastSeen,
		agent.LastHeartbeat,
		agent.Version,
		hardwareJSON,
		agent.UpdatedAt,
		agent.Certificate,
		agent.PrivateKey,
		certInfoJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	return nil
}

// Delete deletes an agent
func (r *AgentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, queries.DeleteAgent, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}

	return nil
}

// List retrieves all agents with optional filters
func (r *AgentRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Agent, error) {
	var status *string
	if s, ok := filters["status"].(string); ok {
		status = &s
	}

	rows, err := r.db.QueryContext(ctx, queries.ListAgents, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agent models.Agent
		var hardwareJSON, certInfoJSON []byte
		var createdByUser models.User

		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Status,
			&agent.LastError,
			&agent.LastSeen,
			&agent.LastHeartbeat,
			&agent.Version,
			&hardwareJSON,
			&agent.CreatedByID,
			&agent.CreatedAt,
			&agent.UpdatedAt,
			&agent.Certificate,
			&agent.PrivateKey,
			&certInfoJSON,
			&createdByUser.ID,
			&createdByUser.Username,
			&createdByUser.Email,
			&createdByUser.Role,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}

		// Unmarshal hardware JSON
		if err := json.Unmarshal(hardwareJSON, &agent.Hardware); err != nil {
			return nil, fmt.Errorf("failed to unmarshal hardware: %w", err)
		}

		// Unmarshal certificate info JSON
		if err := json.Unmarshal(certInfoJSON, &agent.CertificateInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal certificate info: %w", err)
		}

		agent.CreatedBy = &createdByUser
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return agents, nil
}

// UpdateStatus updates an agent's status and last error
func (r *AgentRepository) UpdateStatus(ctx context.Context, id string, status string, lastError *string) error {
	var nullLastError sql.NullString
	if lastError != nil {
		nullLastError = sql.NullString{String: *lastError, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, queries.UpdateAgentStatus, id, status, nullLastError)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}

	return nil
}

// UpdateHeartbeat updates an agent's heartbeat timestamp
func (r *AgentRepository) UpdateHeartbeat(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, queries.UpdateAgentHeartbeat, id)
	if err != nil {
		return fmt.Errorf("failed to update agent heartbeat: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}

	return nil
}

// SaveMetrics saves agent metrics
func (r *AgentRepository) SaveMetrics(ctx context.Context, metrics *models.AgentMetrics) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_metrics (
			agent_id, cpu_usage, gpu_usage, gpu_temp,
			memory_usage, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		metrics.AgentID,
		metrics.CPUUsage,
		metrics.GPUUsage,
		metrics.GPUTemp,
		metrics.MemoryUsage,
		metrics.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save agent metrics: %w", err)
	}

	return nil
}
