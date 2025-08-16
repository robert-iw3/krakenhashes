-- Restore is_small_job column to preset_jobs table
ALTER TABLE preset_jobs ADD COLUMN IF NOT EXISTS is_small_job BOOLEAN DEFAULT FALSE;

-- Restore is_small_job column to job_executions table
ALTER TABLE job_executions ADD COLUMN IF NOT EXISTS is_small_job BOOLEAN DEFAULT FALSE;