-- Add user notification preferences for job completion emails
ALTER TABLE users 
ADD COLUMN notify_on_job_completion BOOLEAN NOT NULL DEFAULT FALSE;

-- Add email tracking to job_executions table
ALTER TABLE job_executions
ADD COLUMN completion_email_sent BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN completion_email_sent_at TIMESTAMP,
ADD COLUMN completion_email_error TEXT;

-- Create index for finding users with notifications enabled
CREATE INDEX idx_users_notify_job_completion ON users (notify_on_job_completion) WHERE notify_on_job_completion = true;

-- Create index for tracking email status on jobs
CREATE INDEX idx_job_executions_email_sent ON job_executions (completion_email_sent, completed_at) 
WHERE completed_at IS NOT NULL;