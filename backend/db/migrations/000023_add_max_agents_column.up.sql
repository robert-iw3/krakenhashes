-- Add max_agents column to job_executions table
ALTER TABLE job_executions 
ADD COLUMN max_agents INTEGER DEFAULT 1 NOT NULL;

-- Add updated_at column to job_executions table if it doesn't exist
ALTER TABLE job_executions 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Create trigger to automatically update updated_at column
CREATE OR REPLACE FUNCTION update_job_executions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop trigger if it exists and create new one
DROP TRIGGER IF EXISTS trigger_update_job_executions_updated_at ON job_executions;
CREATE TRIGGER trigger_update_job_executions_updated_at
    BEFORE UPDATE ON job_executions
    FOR EACH ROW
    EXECUTE FUNCTION update_job_executions_updated_at();

-- Add comment to document the column
COMMENT ON COLUMN job_executions.max_agents IS 'Maximum number of agents that can work on this job execution concurrently';