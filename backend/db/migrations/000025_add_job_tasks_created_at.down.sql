-- Remove created_at column from job_tasks table
ALTER TABLE job_tasks DROP COLUMN IF EXISTS created_at;