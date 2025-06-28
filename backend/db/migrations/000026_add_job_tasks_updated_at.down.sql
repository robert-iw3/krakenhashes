-- Remove trigger first
DROP TRIGGER IF EXISTS update_job_tasks_updated_at ON job_tasks;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remove indexes
DROP INDEX IF EXISTS idx_job_tasks_updated_at;
DROP INDEX IF EXISTS idx_job_tasks_status_updated_at;

-- Remove column
ALTER TABLE job_tasks DROP COLUMN IF EXISTS updated_at;