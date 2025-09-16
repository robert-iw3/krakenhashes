-- Remove sync status tracking from agents table

-- Drop index
DROP INDEX IF EXISTS idx_agents_sync_status;

-- Remove columns
ALTER TABLE agents
DROP COLUMN IF EXISTS sync_status,
DROP COLUMN IF EXISTS sync_completed_at,
DROP COLUMN IF EXISTS sync_started_at,
DROP COLUMN IF EXISTS sync_error,
DROP COLUMN IF EXISTS files_to_sync,
DROP COLUMN IF EXISTS files_synced;

-- Drop enum type
DROP TYPE IF EXISTS agent_sync_status;