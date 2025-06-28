-- Drop the existing constraint to add the new status
ALTER TABLE job_executions DROP CONSTRAINT IF EXISTS valid_status;

-- Add the new constraint with interrupted status
ALTER TABLE job_executions ADD CONSTRAINT valid_status 
CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled', 'interrupted'));

-- Add task timeout setting
INSERT INTO system_settings (key, value, description, data_type)
VALUES ('task_timeout_minutes', '30', 'Maximum time in minutes a task can run without progress updates before being marked as stale', 'integer')
ON CONFLICT (key) DO NOTHING;

-- Add index for finding stale tasks efficiently
CREATE INDEX IF NOT EXISTS idx_job_tasks_status_last_checkpoint 
ON job_tasks(status, last_checkpoint) 
WHERE status IN ('assigned', 'running');