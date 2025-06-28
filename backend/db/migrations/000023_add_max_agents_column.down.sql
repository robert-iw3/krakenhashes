-- Remove trigger and function
DROP TRIGGER IF EXISTS trigger_update_job_executions_updated_at ON job_executions;
DROP FUNCTION IF EXISTS update_job_executions_updated_at();

-- Remove max_agents column from job_executions table
ALTER TABLE job_executions 
DROP COLUMN IF EXISTS max_agents;

-- Note: We don't remove updated_at column as it might be used by other parts of the system