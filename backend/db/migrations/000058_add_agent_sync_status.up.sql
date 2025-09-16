-- Add sync status tracking to agents table
-- This allows the backend to track whether an agent has completed file synchronization
-- and prevent job assignment until sync is complete

-- Add sync_status enum type
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'agent_sync_status') THEN
        CREATE TYPE agent_sync_status AS ENUM ('pending', 'in_progress', 'completed', 'failed');
    END IF;
END$$;

-- Add sync tracking columns to agents table
ALTER TABLE agents
ADD COLUMN IF NOT EXISTS sync_status agent_sync_status DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS sync_completed_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS sync_started_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS sync_error TEXT,
ADD COLUMN IF NOT EXISTS files_to_sync INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS files_synced INTEGER DEFAULT 0;

-- Create index for efficient querying of agents by sync status
CREATE INDEX IF NOT EXISTS idx_agents_sync_status ON agents(sync_status);

-- Update existing active agents to completed status (assume they're already synced)
UPDATE agents
SET sync_status = 'completed',
    sync_completed_at = CURRENT_TIMESTAMP
WHERE status = 'active'
  AND sync_status = 'pending';

COMMENT ON COLUMN agents.sync_status IS 'Current file synchronization status of the agent';
COMMENT ON COLUMN agents.sync_completed_at IS 'Timestamp when the agent last completed file synchronization';
COMMENT ON COLUMN agents.sync_started_at IS 'Timestamp when the current file synchronization started';
COMMENT ON COLUMN agents.sync_error IS 'Error message if sync failed';
COMMENT ON COLUMN agents.files_to_sync IS 'Total number of files that need to be synchronized';
COMMENT ON COLUMN agents.files_synced IS 'Number of files successfully synchronized';