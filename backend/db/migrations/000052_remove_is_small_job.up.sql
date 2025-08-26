-- Remove is_small_job column from preset_jobs table
-- This field is no longer used as the scheduler dynamically determines job splitting
ALTER TABLE preset_jobs DROP COLUMN IF EXISTS is_small_job;

-- Remove is_small_job column from job_executions table if it exists there
ALTER TABLE job_executions DROP COLUMN IF EXISTS is_small_job;