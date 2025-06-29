-- Add owner_id column to agents table to track who the agent is assigned to
ALTER TABLE agents ADD COLUMN owner_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- Add extra_parameters column to agents table to store agent-specific hashcat parameters
ALTER TABLE agents ADD COLUMN extra_parameters TEXT DEFAULT '';

-- Create index on owner_id for better query performance
CREATE INDEX idx_agents_owner_id ON agents(owner_id);

-- Add comment to explain the purpose of these columns
COMMENT ON COLUMN agents.owner_id IS 'The user who owns/manages this agent';
COMMENT ON COLUMN agents.extra_parameters IS 'Agent-specific hashcat parameters (e.g., -w 4 -O)';