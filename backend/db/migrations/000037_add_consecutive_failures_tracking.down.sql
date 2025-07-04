-- Remove indexes
DROP INDEX IF EXISTS idx_agents_consecutive_failures;
DROP INDEX IF EXISTS idx_job_executions_consecutive_failures;

-- Remove columns
ALTER TABLE agents
DROP COLUMN IF EXISTS consecutive_failures;

ALTER TABLE job_executions
DROP COLUMN IF EXISTS consecutive_failures;