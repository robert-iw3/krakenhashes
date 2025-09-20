package repository

import (
	"context"
	"fmt"
)

// HasActiveJobsUsingWordlist checks if there are any active jobs using the specified wordlist
func (r *JobExecutionRepository) HasActiveJobsUsingWordlist(ctx context.Context, wordlistID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM job_executions
			WHERE status NOT IN ('completed', 'cancelled', 'failed')
			AND wordlist_ids ? $1
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, wordlistID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check active jobs using wordlist: %w", err)
	}

	return exists, nil
}

// HasActiveJobsUsingRule checks if there are any active jobs using the specified rule
func (r *JobExecutionRepository) HasActiveJobsUsingRule(ctx context.Context, ruleID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM job_executions
			WHERE status NOT IN ('completed', 'cancelled', 'failed')
			AND rule_ids ? $1
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check active jobs using rule: %w", err)
	}

	return exists, nil
}