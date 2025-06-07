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

// JobTaskRepository handles database operations for job tasks
type JobTaskRepository struct {
	db *db.DB
}

// NewJobTaskRepository creates a new job task repository
func NewJobTaskRepository(db *db.DB) *JobTaskRepository {
	return &JobTaskRepository{db: db}
}

// Create creates a new job task
func (r *JobTaskRepository) Create(ctx context.Context, task *models.JobTask) error {
	query := `
		INSERT INTO job_tasks (
			job_execution_id, agent_id, status, keyspace_start, keyspace_end, 
			keyspace_processed, benchmark_speed, chunk_duration
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, assigned_at`

	err := r.db.QueryRowContext(ctx, query,
		task.JobExecutionID,
		task.AgentID,
		task.Status,
		task.KeyspaceStart,
		task.KeyspaceEnd,
		task.KeyspaceProcessed,
		task.BenchmarkSpeed,
		task.ChunkDuration,
	).Scan(&task.ID, &task.AssignedAt)

	if err != nil {
		return fmt.Errorf("failed to create job task: %w", err)
	}

	return nil
}

// GetByID retrieves a job task by ID
func (r *JobTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobTask, error) {
	query := `
		SELECT 
			jt.id, jt.job_execution_id, jt.agent_id, jt.status,
			jt.keyspace_start, jt.keyspace_end, jt.keyspace_processed,
			jt.benchmark_speed, jt.chunk_duration, jt.assigned_at,
			jt.started_at, jt.completed_at, jt.last_checkpoint, jt.error_message,
			a.name as agent_name
		FROM job_tasks jt
		JOIN agents a ON jt.agent_id = a.id
		WHERE jt.id = $1`

	var task models.JobTask
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
		&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
		&task.BenchmarkSpeed, &task.ChunkDuration, &task.AssignedAt,
		&task.StartedAt, &task.CompletedAt, &task.LastCheckpoint, &task.ErrorMessage,
		&task.AgentName,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job task: %w", err)
	}

	return &task, nil
}

// GetTasksByJobExecution retrieves all tasks for a job execution
func (r *JobTaskRepository) GetTasksByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) ([]models.JobTask, error) {
	query := `
		SELECT 
			jt.id, jt.job_execution_id, jt.agent_id, jt.status,
			jt.keyspace_start, jt.keyspace_end, jt.keyspace_processed,
			jt.benchmark_speed, jt.chunk_duration, jt.assigned_at,
			jt.started_at, jt.completed_at, jt.last_checkpoint, jt.error_message,
			a.name as agent_name
		FROM job_tasks jt
		JOIN agents a ON jt.agent_id = a.id
		WHERE jt.job_execution_id = $1
		ORDER BY jt.keyspace_start ASC`

	rows, err := r.db.QueryContext(ctx, query, jobExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by job execution: %w", err)
	}
	defer rows.Close()

	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
			&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.ChunkDuration, &task.AssignedAt,
			&task.StartedAt, &task.CompletedAt, &task.LastCheckpoint, &task.ErrorMessage,
			&task.AgentName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetActiveTasksByAgent retrieves active tasks for an agent
func (r *JobTaskRepository) GetActiveTasksByAgent(ctx context.Context, agentID int) ([]models.JobTask, error) {
	query := `
		SELECT 
			id, job_execution_id, agent_id, status,
			keyspace_start, keyspace_end, keyspace_processed,
			benchmark_speed, chunk_duration, assigned_at,
			started_at, completed_at, last_checkpoint, error_message
		FROM job_tasks
		WHERE agent_id = $1 AND status IN ('assigned', 'running')
		ORDER BY assigned_at ASC`

	rows, err := r.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks by agent: %w", err)
	}
	defer rows.Close()

	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
			&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.ChunkDuration, &task.AssignedAt,
			&task.StartedAt, &task.CompletedAt, &task.LastCheckpoint, &task.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateStatus updates the status of a job task
func (r *JobTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobTaskStatus) error {
	query := `UPDATE job_tasks SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update job task status: %w", err)
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

// StartTask marks a task as started
func (r *JobTaskRepository) StartTask(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE job_tasks SET status = $1, started_at = $2 WHERE id = $3 AND status = 'assigned'`
	result, err := r.db.ExecContext(ctx, query, models.JobTaskStatusRunning, now, id)
	if err != nil {
		return fmt.Errorf("failed to start job task: %w", err)
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

// UpdateProgress updates the progress of a task
func (r *JobTaskRepository) UpdateProgress(ctx context.Context, id uuid.UUID, keyspaceProcessed int64, benchmarkSpeed *int64) error {
	now := time.Now()
	query := `
		UPDATE job_tasks 
		SET keyspace_processed = $1, benchmark_speed = $2, last_checkpoint = $3 
		WHERE id = $4`
	
	result, err := r.db.ExecContext(ctx, query, keyspaceProcessed, benchmarkSpeed, now, id)
	if err != nil {
		return fmt.Errorf("failed to update job task progress: %w", err)
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

// CompleteTask marks a task as completed
func (r *JobTaskRepository) CompleteTask(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE job_tasks SET status = $1, completed_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, models.JobTaskStatusCompleted, now, id)
	if err != nil {
		return fmt.Errorf("failed to complete job task: %w", err)
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

// FailTask marks a task as failed with an error message
func (r *JobTaskRepository) FailTask(ctx context.Context, id uuid.UUID, errorMessage string) error {
	now := time.Now()
	query := `UPDATE job_tasks SET status = $1, completed_at = $2, error_message = $3 WHERE id = $4`
	result, err := r.db.ExecContext(ctx, query, models.JobTaskStatusFailed, now, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to fail job task: %w", err)
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

// CancelTask marks a task as cancelled
func (r *JobTaskRepository) CancelTask(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE job_tasks SET status = $1, completed_at = $2 WHERE id = $3 AND status IN ('pending', 'assigned', 'running')`
	result, err := r.db.ExecContext(ctx, query, models.JobTaskStatusCancelled, now, id)
	if err != nil {
		return fmt.Errorf("failed to cancel job task: %w", err)
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

// GetNextKeyspaceRange gets the next available keyspace range for a job
func (r *JobTaskRepository) GetNextKeyspaceRange(ctx context.Context, jobExecutionID uuid.UUID) (start int64, end int64, err error) {
	query := `
		SELECT COALESCE(MAX(keyspace_end), 0) as last_end
		FROM job_tasks
		WHERE job_execution_id = $1`

	var lastEnd int64
	err = r.db.QueryRowContext(ctx, query, jobExecutionID).Scan(&lastEnd)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get next keyspace range: %w", err)
	}

	return lastEnd, 0, nil // End will be calculated by the service based on chunk size
}

// GetIncompleteTasksCount returns the number of incomplete tasks for a job
func (r *JobTaskRepository) GetIncompleteTasksCount(ctx context.Context, jobExecutionID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM job_tasks
		WHERE job_execution_id = $1 AND status NOT IN ('completed', 'cancelled')`

	var count int
	err := r.db.QueryRowContext(ctx, query, jobExecutionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get incomplete tasks count: %w", err)
	}

	return count, nil
}