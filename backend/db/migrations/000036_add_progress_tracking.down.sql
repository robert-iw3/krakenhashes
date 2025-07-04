-- Remove progress tracking columns
ALTER TABLE job_executions
DROP COLUMN IF EXISTS overall_progress_percent,
DROP COLUMN IF EXISTS last_progress_update;

ALTER TABLE job_tasks 
DROP COLUMN IF EXISTS progress_percent;

-- Drop indexes
DROP INDEX IF EXISTS idx_job_tasks_progress;
DROP INDEX IF EXISTS idx_job_executions_progress;