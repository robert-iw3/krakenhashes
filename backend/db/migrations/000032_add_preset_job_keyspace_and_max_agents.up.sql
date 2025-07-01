-- Add keyspace column to preset_jobs table
ALTER TABLE preset_jobs 
ADD COLUMN keyspace BIGINT DEFAULT NULL;

-- Add max_agents column to preset_jobs table
ALTER TABLE preset_jobs 
ADD COLUMN max_agents INTEGER DEFAULT 0 NOT NULL;

-- Add comments to document the columns
COMMENT ON COLUMN preset_jobs.keyspace IS 'Pre-calculated total keyspace for this preset job configuration';
COMMENT ON COLUMN preset_jobs.max_agents IS 'Maximum number of agents that can work on executions of this preset job concurrently (0 = unlimited)';

-- Create index on keyspace for faster queries
CREATE INDEX idx_preset_jobs_keyspace ON preset_jobs(keyspace) WHERE keyspace IS NOT NULL;