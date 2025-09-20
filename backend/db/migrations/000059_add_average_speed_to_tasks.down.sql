-- Remove average_speed column from job_tasks table
ALTER TABLE job_tasks
DROP COLUMN IF EXISTS average_speed;