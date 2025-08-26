-- Add is_enabled column to agents table to control whether agent can accept jobs
ALTER TABLE agents ADD COLUMN is_enabled BOOLEAN NOT NULL DEFAULT true;

-- Create index for better query performance when filtering enabled agents
CREATE INDEX idx_agents_is_enabled ON agents(is_enabled);

-- Add comment to explain the purpose
COMMENT ON COLUMN agents.is_enabled IS 'Controls whether the agent can accept new jobs. False = maintenance mode';