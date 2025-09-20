-- Add average_speed column to job_tasks table to store time-weighted average hash rate
ALTER TABLE job_tasks
ADD COLUMN average_speed BIGINT;

-- Add comment to clarify the purpose
COMMENT ON COLUMN job_tasks.average_speed IS 'Time-weighted average hash rate calculated when task completes';