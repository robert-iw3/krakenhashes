-- Drop index
DROP INDEX IF EXISTS idx_agents_is_enabled;

-- Remove column
ALTER TABLE agents DROP COLUMN IF EXISTS is_enabled;