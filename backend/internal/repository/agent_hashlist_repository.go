package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
)

// AgentHashlistRepository handles database operations for agent hashlist distribution
type AgentHashlistRepository struct {
	db *db.DB
}

// NewAgentHashlistRepository creates a new agent hashlist repository
func NewAgentHashlistRepository(db *db.DB) *AgentHashlistRepository {
	return &AgentHashlistRepository{db: db}
}

// CreateOrUpdate creates or updates an agent hashlist distribution record
func (r *AgentHashlistRepository) CreateOrUpdate(ctx context.Context, agentHashlist *models.AgentHashlist) error {
	query := `
		INSERT INTO agent_hashlists (agent_id, hashlist_id, file_path, file_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (agent_id, hashlist_id)
		DO UPDATE SET 
			file_path = $3,
			file_hash = $4,
			downloaded_at = CURRENT_TIMESTAMP,
			last_used_at = CURRENT_TIMESTAMP
		RETURNING id, downloaded_at, last_used_at`

	err := r.db.QueryRowContext(ctx, query,
		agentHashlist.AgentID,
		agentHashlist.HashlistID,
		agentHashlist.FilePath,
		agentHashlist.FileHash,
	).Scan(&agentHashlist.ID, &agentHashlist.DownloadedAt, &agentHashlist.LastUsedAt)

	if err != nil {
		return fmt.Errorf("failed to create or update agent hashlist: %w", err)
	}

	return nil
}

// GetByAgentAndHashlist retrieves an agent hashlist record
func (r *AgentHashlistRepository) GetByAgentAndHashlist(ctx context.Context, agentID int, hashlistID int64) (*models.AgentHashlist, error) {
	query := `
		SELECT id, agent_id, hashlist_id, file_path, downloaded_at, last_used_at, file_hash
		FROM agent_hashlists
		WHERE agent_id = $1 AND hashlist_id = $2`

	var agentHashlist models.AgentHashlist
	err := r.db.QueryRowContext(ctx, query, agentID, hashlistID).Scan(
		&agentHashlist.ID,
		&agentHashlist.AgentID,
		&agentHashlist.HashlistID,
		&agentHashlist.FilePath,
		&agentHashlist.DownloadedAt,
		&agentHashlist.LastUsedAt,
		&agentHashlist.FileHash,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent hashlist: %w", err)
	}

	return &agentHashlist, nil
}

// GetHashlistsByAgent retrieves all hashlists for an agent
func (r *AgentHashlistRepository) GetHashlistsByAgent(ctx context.Context, agentID int) ([]models.AgentHashlist, error) {
	query := `
		SELECT ah.id, ah.agent_id, ah.hashlist_id, ah.file_path, ah.downloaded_at, ah.last_used_at, ah.file_hash
		FROM agent_hashlists ah
		WHERE ah.agent_id = $1
		ORDER BY ah.last_used_at DESC`

	rows, err := r.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashlists by agent: %w", err)
	}
	defer rows.Close()

	var agentHashlists []models.AgentHashlist
	for rows.Next() {
		var agentHashlist models.AgentHashlist
		err := rows.Scan(
			&agentHashlist.ID,
			&agentHashlist.AgentID,
			&agentHashlist.HashlistID,
			&agentHashlist.FilePath,
			&agentHashlist.DownloadedAt,
			&agentHashlist.LastUsedAt,
			&agentHashlist.FileHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent hashlist: %w", err)
		}
		agentHashlists = append(agentHashlists, agentHashlist)
	}

	return agentHashlists, nil
}

// UpdateLastUsed updates the last used timestamp for an agent hashlist
func (r *AgentHashlistRepository) UpdateLastUsed(ctx context.Context, agentID int, hashlistID int64) error {
	query := `
		UPDATE agent_hashlists 
		SET last_used_at = CURRENT_TIMESTAMP 
		WHERE agent_id = $1 AND hashlist_id = $2`

	result, err := r.db.ExecContext(ctx, query, agentID, hashlistID)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// CleanupOldHashlists removes hashlists that haven't been used within the retention period
func (r *AgentHashlistRepository) CleanupOldHashlists(ctx context.Context, retentionPeriod time.Duration) ([]models.AgentHashlist, error) {
	cutoffTime := time.Now().Add(-retentionPeriod)
	
	// First, get the records to be deleted for cleanup purposes
	selectQuery := `
		SELECT id, agent_id, hashlist_id, file_path, downloaded_at, last_used_at, file_hash
		FROM agent_hashlists
		WHERE last_used_at < $1`

	rows, err := r.db.QueryContext(ctx, selectQuery, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to select old hashlists: %w", err)
	}
	defer rows.Close()

	var deletedHashlists []models.AgentHashlist
	for rows.Next() {
		var agentHashlist models.AgentHashlist
		err := rows.Scan(
			&agentHashlist.ID,
			&agentHashlist.AgentID,
			&agentHashlist.HashlistID,
			&agentHashlist.FilePath,
			&agentHashlist.DownloadedAt,
			&agentHashlist.LastUsedAt,
			&agentHashlist.FileHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan old hashlist: %w", err)
		}
		deletedHashlists = append(deletedHashlists, agentHashlist)
	}

	// Now delete the records
	deleteQuery := `DELETE FROM agent_hashlists WHERE last_used_at < $1`
	_, err = r.db.ExecContext(ctx, deleteQuery, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old hashlists: %w", err)
	}

	return deletedHashlists, nil
}

// CleanupAgentHashlists removes all hashlists for a specific agent (used when agent is removed)
func (r *AgentHashlistRepository) CleanupAgentHashlists(ctx context.Context, agentID int) ([]models.AgentHashlist, error) {
	// First, get the records to be deleted for cleanup purposes
	selectQuery := `
		SELECT id, agent_id, hashlist_id, file_path, downloaded_at, last_used_at, file_hash
		FROM agent_hashlists
		WHERE agent_id = $1`

	rows, err := r.db.QueryContext(ctx, selectQuery, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to select agent hashlists: %w", err)
	}
	defer rows.Close()

	var deletedHashlists []models.AgentHashlist
	for rows.Next() {
		var agentHashlist models.AgentHashlist
		err := rows.Scan(
			&agentHashlist.ID,
			&agentHashlist.AgentID,
			&agentHashlist.HashlistID,
			&agentHashlist.FilePath,
			&agentHashlist.DownloadedAt,
			&agentHashlist.LastUsedAt,
			&agentHashlist.FileHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent hashlist: %w", err)
		}
		deletedHashlists = append(deletedHashlists, agentHashlist)
	}

	// Now delete the records
	deleteQuery := `DELETE FROM agent_hashlists WHERE agent_id = $1`
	_, err = r.db.ExecContext(ctx, deleteQuery, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent hashlists: %w", err)
	}

	return deletedHashlists, nil
}

// IsHashlistCurrentForAgent checks if the agent has the current version of the hashlist
func (r *AgentHashlistRepository) IsHashlistCurrentForAgent(ctx context.Context, agentID int, hashlistID int64, currentFileHash string) (bool, error) {
	query := `
		SELECT file_hash
		FROM agent_hashlists
		WHERE agent_id = $1 AND hashlist_id = $2`

	var agentFileHash *string
	err := r.db.QueryRowContext(ctx, query, agentID, hashlistID).Scan(&agentFileHash)
	
	if err == sql.ErrNoRows {
		return false, nil // Agent doesn't have this hashlist
	}
	if err != nil {
		return false, fmt.Errorf("failed to check hashlist currency: %w", err)
	}

	if agentFileHash == nil {
		return false, nil // No hash stored, assume outdated
	}

	return *agentFileHash == currentFileHash, nil
}

// GetHashlistDistribution returns which agents have a specific hashlist
func (r *AgentHashlistRepository) GetHashlistDistribution(ctx context.Context, hashlistID int64) ([]models.AgentHashlist, error) {
	query := `
		SELECT ah.id, ah.agent_id, ah.hashlist_id, ah.file_path, ah.downloaded_at, ah.last_used_at, ah.file_hash
		FROM agent_hashlists ah
		WHERE ah.hashlist_id = $1
		ORDER BY ah.last_used_at DESC`

	rows, err := r.db.QueryContext(ctx, query, hashlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashlist distribution: %w", err)
	}
	defer rows.Close()

	var agentHashlists []models.AgentHashlist
	for rows.Next() {
		var agentHashlist models.AgentHashlist
		err := rows.Scan(
			&agentHashlist.ID,
			&agentHashlist.AgentID,
			&agentHashlist.HashlistID,
			&agentHashlist.FilePath,
			&agentHashlist.DownloadedAt,
			&agentHashlist.LastUsedAt,
			&agentHashlist.FileHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent hashlist: %w", err)
		}
		agentHashlists = append(agentHashlists, agentHashlist)
	}

	return agentHashlists, nil
}