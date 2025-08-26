-- Drop the index
DROP INDEX IF EXISTS idx_job_executions_created_by;

-- Remove the created_by column
ALTER TABLE job_executions DROP COLUMN created_by;