package repository

import (
	"context"
	"fmt"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
)

// GetAllTasksByJobExecution retrieves ALL tasks for a job execution without pagination
func (r *JobTaskRepository) GetAllTasksByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) ([]models.JobTask, error) {
	query := `
		SELECT
			id, job_execution_id, agent_id, status, keyspace_start, keyspace_end,
			keyspace_processed, benchmark_speed, chunk_duration,
			COALESCE(crack_count, 0) as crack_count,
			COALESCE(detailed_status, 'pending') as detailed_status,
			COALESCE(retry_count, 0) as retry_count,
			error_message,
			created_at, started_at, completed_at, updated_at,
			effective_keyspace_start, effective_keyspace_end, effective_keyspace_processed,
			rule_start_index, rule_end_index, is_rule_split_task,
			progress_percent, average_speed
		FROM job_tasks
		WHERE job_execution_id = $1
		ORDER BY
			CASE
				WHEN status = 'completed' THEN completed_at
				ELSE created_at
			END DESC`

	rows, err := r.db.QueryContext(ctx, query, jobExecutionID)
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
			&task.EffectiveKeyspaceStart, &task.EffectiveKeyspaceEnd, &task.EffectiveKeyspaceProcessed,
			&task.RuleStartIndex, &task.RuleEndIndex, &task.IsRuleSplitTask,
			&task.ProgressPercent, &task.AverageSpeed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

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
			created_at, started_at, completed_at, updated_at,
			effective_keyspace_start, effective_keyspace_end, effective_keyspace_processed,
			rule_start_index, rule_end_index, is_rule_split_task,
			progress_percent
		FROM job_tasks
		WHERE job_execution_id = $1
		ORDER BY
			CASE
				WHEN status = 'completed' THEN completed_at
				ELSE created_at
			END DESC
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
			&task.EffectiveKeyspaceStart, &task.EffectiveKeyspaceEnd, &task.EffectiveKeyspaceProcessed,
			&task.RuleStartIndex, &task.RuleEndIndex, &task.IsRuleSplitTask,
			&task.ProgressPercent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetActiveTasksByJobExecution retrieves all active tasks for a job execution
func (r *JobTaskRepository) GetActiveTasksByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) ([]models.JobTask, error) {
	query := `
		SELECT
			id, job_execution_id, agent_id, status, keyspace_start, keyspace_end,
			keyspace_processed, benchmark_speed, average_speed, chunk_duration,
			COALESCE(crack_count, 0) as crack_count,
			COALESCE(detailed_status, 'pending') as detailed_status,
			COALESCE(retry_count, 0) as retry_count,
			error_message,
			created_at, started_at, completed_at, updated_at,
			effective_keyspace_start, effective_keyspace_end, effective_keyspace_processed,
			rule_start_index, rule_end_index, is_rule_split_task,
			progress_percent, assigned_at, last_checkpoint, chunk_number
		FROM job_tasks
		WHERE job_execution_id = $1
		AND status IN ('running', 'assigned', 'pending', 'reconnect_pending')
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, jobExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks for job execution: %w", err)
	}
	defer rows.Close()

	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status,
			&task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.AverageSpeed, &task.ChunkDuration,
			&task.CrackCount, &task.DetailedStatus, &task.RetryCount,
			&task.ErrorMessage,
			&task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.UpdatedAt,
			&task.EffectiveKeyspaceStart, &task.EffectiveKeyspaceEnd, &task.EffectiveKeyspaceProcessed,
			&task.RuleStartIndex, &task.RuleEndIndex, &task.IsRuleSplitTask,
			&task.ProgressPercent, &task.AssignedAt, &task.LastCheckpoint, &task.ChunkNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

/*
// GetTasksByStatuses retrieves all tasks with the specified statuses
// MOVED TO job_task_repository.go
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
*/

/*
// GetActiveTasksCount returns the number of active (assigned or running) tasks for a job execution
// MOVED TO job_task_repository.go
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
*/

/*
// GetTaskCountByJobExecution returns the total number of tasks for a job execution
// MOVED TO job_task_repository.go
func (r *JobTaskRepository) GetTaskCountByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM job_tasks WHERE job_execution_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, jobExecutionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count for job execution: %w", err)
	}
	return count, nil
}
*/

// AreAllTasksComplete checks if all tasks for a job execution are complete
func (r *JobTaskRepository) AreAllTasksComplete(ctx context.Context, jobExecutionID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) = 0
		FROM job_tasks
		WHERE job_execution_id = $1
		AND status NOT IN ($2, $3, $4, $5)`

	var allComplete bool
	err := r.db.QueryRowContext(ctx, query,
		jobExecutionID,
		models.JobTaskStatusCompleted,
		models.JobTaskStatusFailed,
		models.JobTaskStatusCancelled,
		"completed", // Handle both constants and string values
	).Scan(&allComplete)

	if err != nil {
		return false, fmt.Errorf("failed to check if all tasks complete: %w", err)
	}

	return allComplete, nil
}

// GetPendingTasksByJobExecution retrieves all pending tasks for a specific job execution
func (r *JobTaskRepository) GetPendingTasksByJobExecution(ctx context.Context, jobExecutionID uuid.UUID) ([]models.JobTask, error) {
	query := `
		SELECT id, job_execution_id, agent_id, status, priority, attack_cmd,
			keyspace_start, keyspace_end, keyspace_processed, benchmark_speed,
			chunk_duration, created_at, assigned_at, started_at, completed_at,
			updated_at, last_checkpoint, error_message, crack_count,
			detailed_status, retry_count, rule_start_index, rule_end_index,
			rule_chunk_path, is_rule_split_task
		FROM job_tasks
		WHERE job_execution_id = $1 AND status = $2
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, jobExecutionID, models.JobTaskStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.JobTask
	for rows.Next() {
		var task models.JobTask
		err := rows.Scan(
			&task.ID, &task.JobExecutionID, &task.AgentID, &task.Status, &task.Priority,
			&task.AttackCmd, &task.KeyspaceStart, &task.KeyspaceEnd, &task.KeyspaceProcessed,
			&task.BenchmarkSpeed, &task.ChunkDuration, &task.CreatedAt, &task.AssignedAt,
			&task.StartedAt, &task.CompletedAt, &task.UpdatedAt, &task.LastCheckpoint,
			&task.ErrorMessage, &task.CrackCount, &task.DetailedStatus, &task.RetryCount,
			&task.RuleStartIndex, &task.RuleEndIndex, &task.RuleChunkPath, &task.IsRuleSplitTask,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Update updates a job task
func (r *JobTaskRepository) Update(ctx context.Context, task *models.JobTask) error {
	query := `
		UPDATE job_tasks SET
			agent_id = $1, status = $2, priority = $3, attack_cmd = $4,
			keyspace_start = $5, keyspace_end = $6, keyspace_processed = $7,
			benchmark_speed = $8, chunk_duration = $9, assigned_at = $10,
			started_at = $11, completed_at = $12, updated_at = $13,
			last_checkpoint = $14, error_message = $15, crack_count = $16,
			detailed_status = $17, retry_count = $18
		WHERE id = $19`

	_, err := r.db.ExecContext(ctx, query,
		task.AgentID, task.Status, task.Priority, task.AttackCmd,
		task.KeyspaceStart, task.KeyspaceEnd, task.KeyspaceProcessed,
		task.BenchmarkSpeed, task.ChunkDuration, task.AssignedAt,
		task.StartedAt, task.CompletedAt, task.UpdatedAt,
		task.LastCheckpoint, task.ErrorMessage, task.CrackCount,
		task.DetailedStatus, task.RetryCount, task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}
