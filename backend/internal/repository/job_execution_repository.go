package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
)

// JobExecutionRepository handles database operations for job executions
type JobExecutionRepository struct {
	db *db.DB
}

// NewJobExecutionRepository creates a new job execution repository
func NewJobExecutionRepository(db *db.DB) *JobExecutionRepository {
	return &JobExecutionRepository{db: db}
}

// Create creates a new job execution
func (r *JobExecutionRepository) Create(ctx context.Context, exec *models.JobExecution) error {
	query := `
		INSERT INTO job_executions (preset_job_id, hashlist_id, status, priority, max_agents, attack_mode, total_keyspace, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query,
		exec.PresetJobID,
		exec.HashlistID,
		exec.Status,
		exec.Priority,
		exec.MaxAgents,
		exec.AttackMode,
		exec.TotalKeyspace,
		exec.CreatedBy,
	).Scan(&exec.ID, &exec.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create job execution: %w", err)
	}

	return nil
}

// GetByID retrieves a job execution by ID
func (r *JobExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	query := `
		SELECT 
			je.id, je.preset_job_id, je.hashlist_id, je.status, je.priority, COALESCE(je.max_agents, 0) as max_agents,
			je.total_keyspace, je.processed_keyspace, je.attack_mode, je.created_by,
			je.created_at, je.started_at, je.completed_at, je.error_message, je.interrupted_by,
			pj.name as preset_job_name,
			h.name as hashlist_name,
			h.total_hashes as total_hashes,
			h.cracked_hashes as cracked_hashes
		FROM job_executions je
		JOIN preset_jobs pj ON je.preset_job_id = pj.id
		JOIN hashlists h ON je.hashlist_id = h.id
		WHERE je.id = $1`

	var exec models.JobExecution
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&exec.ID, &exec.PresetJobID, &exec.HashlistID, &exec.Status, &exec.Priority, &exec.MaxAgents,
		&exec.TotalKeyspace, &exec.ProcessedKeyspace, &exec.AttackMode, &exec.CreatedBy,
		&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt, &exec.ErrorMessage, &exec.InterruptedBy,
		&exec.PresetJobName, &exec.HashlistName, &exec.TotalHashes, &exec.CrackedHashes,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job execution: %w", err)
	}

	return &exec, nil
}

// GetPendingJobs retrieves pending jobs ordered by priority and creation time
func (r *JobExecutionRepository) GetPendingJobs(ctx context.Context) ([]models.JobExecution, error) {
	query := `
		SELECT 
			je.id, je.preset_job_id, je.hashlist_id, je.status, je.priority,
			je.total_keyspace, je.processed_keyspace, je.attack_mode, je.created_by,
			je.created_at, je.started_at, je.completed_at, je.error_message, je.interrupted_by,
			je.max_agents, je.updated_at,
			pj.name as preset_job_name,
			h.name as hashlist_name,
			h.total_hashes as total_hashes,
			h.cracked_hashes as cracked_hashes
		FROM job_executions je
		JOIN preset_jobs pj ON je.preset_job_id = pj.id
		JOIN hashlists h ON je.hashlist_id = h.id
		WHERE je.status = 'pending'
		ORDER BY je.priority DESC, je.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}
	defer rows.Close()

	var executions []models.JobExecution
	for rows.Next() {
		var exec models.JobExecution
		err := rows.Scan(
			&exec.ID, &exec.PresetJobID, &exec.HashlistID, &exec.Status, &exec.Priority,
			&exec.TotalKeyspace, &exec.ProcessedKeyspace, &exec.AttackMode, &exec.CreatedBy,
			&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt, &exec.ErrorMessage, &exec.InterruptedBy,
			&exec.MaxAgents, &exec.UpdatedAt,
			&exec.PresetJobName, &exec.HashlistName, &exec.TotalHashes, &exec.CrackedHashes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// GetRunningJobs retrieves all currently running jobs
func (r *JobExecutionRepository) GetRunningJobs(ctx context.Context) ([]models.JobExecution, error) {
	query := `
		SELECT 
			je.id, je.preset_job_id, je.hashlist_id, je.status, je.priority,
			je.total_keyspace, je.processed_keyspace, je.attack_mode,
			je.created_at, je.started_at, je.completed_at, je.error_message, je.interrupted_by
		FROM job_executions je
		WHERE je.status = 'running'`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get running jobs: %w", err)
	}
	defer rows.Close()

	var executions []models.JobExecution
	for rows.Next() {
		var exec models.JobExecution
		err := rows.Scan(
			&exec.ID, &exec.PresetJobID, &exec.HashlistID, &exec.Status, &exec.Priority,
			&exec.TotalKeyspace, &exec.ProcessedKeyspace, &exec.AttackMode,
			&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt, &exec.ErrorMessage, &exec.InterruptedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// GetJobsByStatus retrieves all job executions with the specified status
func (r *JobExecutionRepository) GetJobsByStatus(ctx context.Context, status models.JobExecutionStatus) ([]models.JobExecution, error) {
	query := `
		SELECT 
			id, hashlist_id, preset_job_id, priority, 
			status, started_at, completed_at, created_at, updated_at,
			attack_mode, total_keyspace, processed_keyspace
		FROM job_executions
		WHERE status = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}
	defer rows.Close()

	var jobs []models.JobExecution
	for rows.Next() {
		var job models.JobExecution
		err := rows.Scan(
			&job.ID, &job.HashlistID, &job.PresetJobID, &job.Priority,
			&job.Status, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
			&job.AttackMode, &job.TotalKeyspace, &job.ProcessedKeyspace,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateStatus updates the status of a job execution
func (r *JobExecutionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobExecutionStatus) error {
	query := `UPDATE job_executions SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update job execution status: %w", err)
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

// UpdateErrorMessage updates the error message for a job execution
func (r *JobExecutionRepository) UpdateErrorMessage(ctx context.Context, id uuid.UUID, errorMessage string) error {
	query := `UPDATE job_executions SET error_message = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update job execution error message: %w", err)
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

// UpdateProgress updates the processed keyspace for a job execution
func (r *JobExecutionRepository) UpdateProgress(ctx context.Context, id uuid.UUID, processedKeyspace int64) error {
	query := `UPDATE job_executions SET processed_keyspace = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, processedKeyspace, id)
	if err != nil {
		return fmt.Errorf("failed to update job execution progress: %w", err)
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

// UpdateCrackedCount updates the cracked_hashes count for a job execution
// DEPRECATED: This method is deprecated as cracked counts are now tracked at the hashlist level
func (r *JobExecutionRepository) UpdateCrackedCount(ctx context.Context, id uuid.UUID, crackedCount int) error {
	// This method is deprecated and should not be used
	// The job_executions table does not have a cracked_hashes column
	// Cracked counts are tracked on the hashlists table
	return fmt.Errorf("UpdateCrackedCount is deprecated - cracked counts are tracked on hashlists table")
}

// StartExecution marks a job execution as started
func (r *JobExecutionRepository) StartExecution(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE job_executions SET status = $1, started_at = $2 WHERE id = $3 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, query, models.JobExecutionStatusRunning, now, id)
	if err != nil {
		return fmt.Errorf("failed to start job execution: %w", err)
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

// CompleteExecution marks a job execution as completed
func (r *JobExecutionRepository) CompleteExecution(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE job_executions SET status = $1, completed_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, models.JobExecutionStatusCompleted, now, id)
	if err != nil {
		return fmt.Errorf("failed to complete job execution: %w", err)
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

// FailExecution marks a job execution as failed with an error message
func (r *JobExecutionRepository) FailExecution(ctx context.Context, id uuid.UUID, errorMessage string) error {
	now := time.Now()
	query := `UPDATE job_executions SET status = $1, completed_at = $2, error_message = $3 WHERE id = $4`
	result, err := r.db.ExecContext(ctx, query, models.JobExecutionStatusFailed, now, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to fail job execution: %w", err)
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

// InterruptExecution marks a job as interrupted by another job
func (r *JobExecutionRepository) InterruptExecution(ctx context.Context, id uuid.UUID, interruptingJobID uuid.UUID) error {
	query := `UPDATE job_executions SET status = $1, interrupted_by = $2 WHERE id = $3 AND status = 'running'`
	result, err := r.db.ExecContext(ctx, query, models.JobExecutionStatusPaused, interruptingJobID, id)
	if err != nil {
		return fmt.Errorf("failed to interrupt job execution: %w", err)
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

// GetInterruptibleJobs retrieves running jobs that can be interrupted
func (r *JobExecutionRepository) GetInterruptibleJobs(ctx context.Context, priority int) ([]models.JobExecution, error) {
	query := `
		SELECT 
			je.id, je.preset_job_id, je.hashlist_id, je.status, je.priority,
			je.total_keyspace, je.processed_keyspace, je.attack_mode,
			je.created_at, je.started_at, je.completed_at, je.error_message, je.interrupted_by
		FROM job_executions je
		JOIN preset_jobs pj ON je.preset_job_id = pj.id
		WHERE je.status = 'running' 
		AND je.priority < $1
		AND pj.allow_high_priority_override = true`

	rows, err := r.db.QueryContext(ctx, query, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to get interruptible jobs: %w", err)
	}
	defer rows.Close()

	var executions []models.JobExecution
	for rows.Next() {
		var exec models.JobExecution
		err := rows.Scan(
			&exec.ID, &exec.PresetJobID, &exec.HashlistID, &exec.Status, &exec.Priority,
			&exec.TotalKeyspace, &exec.ProcessedKeyspace, &exec.AttackMode,
			&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt, &exec.ErrorMessage, &exec.InterruptedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// ClearError clears the error message for a job execution
func (r *JobExecutionRepository) ClearError(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE job_executions SET error_message = NULL WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to clear job execution error: %w", err)
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