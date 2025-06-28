-- Remove index for stale tasks
DROP INDEX IF EXISTS idx_job_tasks_status_last_checkpoint;

-- Remove task timeout setting
DELETE FROM system_settings WHERE key = 'task_timeout_minutes';

-- Restore original constraint without interrupted status
ALTER TABLE job_executions DROP CONSTRAINT IF EXISTS valid_status;
ALTER TABLE job_executions ADD CONSTRAINT valid_status 
CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled'));