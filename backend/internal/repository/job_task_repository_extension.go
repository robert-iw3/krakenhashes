package repository

import (
	"context"
	"fmt"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
)

// GetTasksByJobExecutionWithPagination retrieves tasks for a job execution with pagination
func (r *JobTaskRepository) GetTasksByJobExecutionWithPagination(ctx context.Context, jobExecutionID uuid.UUID, limit, offset int) ([]models.JobTask, error) {
	query := `
		SELECT 
			id, job_execution_id, agent_id, status, keyspace_start, keyspace_end,
			keyspace_processed, benchmark_speed, chunk_duration, 
			COALESCE(crack_count, 0) as crack_count,
			COALESCE(detailed_status, 'pending') as detailed_status,
			COALESCE(retry_count, 0) as retry_count,
			error_message,
			created_at, started_at, completed_at, updated_at
		FROM job_tasks
		WHERE job_execution_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, jobExecutionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for job execution: %w", err)
	}
	defer rows.Close()

	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
			&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.ChunkDuration,
			&task.CrackCount, &task.DetailedStatus, &task.RetryCount,
			&task.ErrorMessage,
			&task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTasksByStatuses retrieves all tasks with the specified statuses
func (r *JobTaskRepository) GetTasksByStatuses(ctx context.Context, statuses []string) ([]models.JobTask, error) {
	if len(statuses) == 0 {
		return []models.JobTask{}, nil
	}
	
	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		args[i] = status
	}
	
	query := fmt.Sprintf(`
		SELECT 
			id, job_execution_id, agent_id, status, keyspace_start, keyspace_end,
			keyspace_processed, benchmark_speed, chunk_duration, 
			COALESCE(crack_count, 0) as crack_count,
			COALESCE(detailed_status, 'pending') as detailed_status,
			COALESCE(retry_count, 0) as retry_count,
			error_message,
			created_at, started_at, completed_at, updated_at,
			last_checkpoint
		FROM job_tasks
		WHERE status IN (%s)
		ORDER BY created_at ASC`, placeholders)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by statuses: %w", err)
	}
	defer rows.Close()
	
	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
			&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.ChunkDuration,
			&task.CrackCount, &task.DetailedStatus, &task.RetryCount,
			&task.ErrorMessage,
			&task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.UpdatedAt,
			&task.LastCheckpoint,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// GetActiveTasksCount returns the number of active (assigned or running) tasks for a job execution
func (r *JobTaskRepository) GetActiveTasksCount(ctx context.Context, jobExecutionID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM job_tasks
		WHERE job_execution_id = $1
		AND status IN ('assigned', 'running')`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, jobExecutionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active tasks count: %w", err)
	}
	
	return count, nil
}

// GetTaskCountByJobExecution returns the total number of tasks for a job execution
func (r *JobTaskRepository) GetTaskCountByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM job_tasks WHERE job_execution_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, jobExecutionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count for job execution: %w", err)
	}
	return count, nil
}