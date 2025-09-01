-- Drop indexes
DROP INDEX IF EXISTS idx_users_notify_job_completion;
DROP INDEX IF EXISTS idx_job_executions_email_sent;

-- Remove email tracking from job_executions
ALTER TABLE job_executions
DROP COLUMN IF EXISTS completion_email_sent,
DROP COLUMN IF EXISTS completion_email_sent_at,
DROP COLUMN IF EXISTS completion_email_error;

-- Remove user notification preferences
ALTER TABLE users 
DROP COLUMN IF EXISTS notify_on_job_completion;