-- Add updated_at column to job_tasks table for proper stale task detection
ALTER TABLE job_tasks 
ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Set initial values for existing records
UPDATE job_tasks 
SET updated_at = COALESCE(completed_at, started_at, assigned_at, created_at, NOW());

-- Create trigger to automatically update the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_job_tasks_updated_at 
BEFORE UPDATE ON job_tasks 
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

-- Create index for efficient stale task queries
CREATE INDEX IF NOT EXISTS idx_job_tasks_updated_at ON job_tasks(updated_at);
CREATE INDEX IF NOT EXISTS idx_job_tasks_status_updated_at ON job_tasks(status, updated_at);

-- Add comment explaining the column
COMMENT ON COLUMN job_tasks.updated_at IS 'Timestamp of last update, used for detecting stale tasks';