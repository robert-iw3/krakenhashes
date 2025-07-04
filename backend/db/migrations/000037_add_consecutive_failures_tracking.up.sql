-- Add consecutive failures tracking to job_executions
ALTER TABLE job_executions
ADD COLUMN consecutive_failures INTEGER DEFAULT 0;

-- Add consecutive failures tracking to agents
ALTER TABLE agents
ADD COLUMN consecutive_failures INTEGER DEFAULT 0;

-- Add index for finding jobs with high failure rates
CREATE INDEX idx_job_executions_consecutive_failures ON job_executions(consecutive_failures) WHERE consecutive_failures > 0;

-- Add index for finding unhealthy agents
CREATE INDEX idx_agents_consecutive_failures ON agents(consecutive_failures) WHERE consecutive_failures > 0;