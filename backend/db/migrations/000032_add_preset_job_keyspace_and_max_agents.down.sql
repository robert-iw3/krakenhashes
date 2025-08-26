-- Drop index
DROP INDEX IF EXISTS idx_preset_jobs_keyspace;

-- Drop columns
ALTER TABLE preset_jobs 
DROP COLUMN IF EXISTS keyspace,
DROP COLUMN IF EXISTS max_agents;