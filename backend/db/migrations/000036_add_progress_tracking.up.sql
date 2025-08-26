-- Add progress tracking to job_tasks
ALTER TABLE job_tasks 
ADD COLUMN progress_percent NUMERIC(5,2) DEFAULT 0 CHECK (progress_percent >= 0 AND progress_percent <= 100);

-- Add overall progress to job_executions
ALTER TABLE job_executions
ADD COLUMN overall_progress_percent NUMERIC(5,2) DEFAULT 0 CHECK (overall_progress_percent >= 0 AND overall_progress_percent <= 100),
ADD COLUMN last_progress_update TIMESTAMP;

-- Create index for performance
CREATE INDEX idx_job_tasks_progress ON job_tasks(job_execution_id, progress_percent);
CREATE INDEX idx_job_executions_progress ON job_executions(overall_progress_percent);

-- Update existing tasks to calculate progress_percent if possible
-- This is safe to run as it only updates tasks that are in progress
UPDATE job_tasks 
SET progress_percent = CASE 
    WHEN (keyspace_end - keyspace_start) > 0 
    THEN LEAST(100, (keyspace_processed::numeric / (keyspace_end - keyspace_start)::numeric) * 100)
    ELSE 0
END
WHERE status = 'running' AND (keyspace_end - keyspace_start) > 0;