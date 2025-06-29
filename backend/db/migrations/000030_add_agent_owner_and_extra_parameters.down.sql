-- Drop index
DROP INDEX IF EXISTS idx_agents_owner_id;

-- Remove columns
ALTER TABLE agents DROP COLUMN IF EXISTS extra_parameters;
ALTER TABLE agents DROP COLUMN IF EXISTS owner_id;