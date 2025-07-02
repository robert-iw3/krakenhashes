-- Add created_by column to job_executions table to track who created the job
ALTER TABLE job_executions 
ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Add comment to document the column
COMMENT ON COLUMN job_executions.created_by IS 'User who created/initiated this job execution';

-- Create index for faster lookups
CREATE INDEX idx_job_executions_created_by ON job_executions(created_by);