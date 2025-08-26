-- Remove indexes
DROP INDEX IF EXISTS idx_job_tasks_detailed_status;
DROP INDEX IF EXISTS idx_job_tasks_retry_count;

-- Remove new system settings
DELETE FROM system_settings 
WHERE key IN ('job_refresh_interval_seconds', 'max_chunk_retry_attempts', 'jobs_per_page_default');

-- Restore original constraint
ALTER TABLE job_tasks 
DROP CONSTRAINT valid_task_status;

ALTER TABLE job_tasks 
ADD CONSTRAINT valid_task_status CHECK (status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled'));

-- Remove new columns from job_tasks table
ALTER TABLE job_tasks 
DROP COLUMN crack_count,
DROP COLUMN detailed_status,
DROP COLUMN retry_count;