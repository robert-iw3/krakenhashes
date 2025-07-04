package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// UpdateConsecutiveFailures updates the consecutive failures count for an agent
func (r *AgentRepository) UpdateConsecutiveFailures(ctx context.Context, agentID int, count int) error {
	query := `UPDATE agents SET consecutive_failures = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, count, agentID)
	if err != nil {
		return fmt.Errorf("failed to update agent consecutive failures: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}