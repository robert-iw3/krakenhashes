-- Create agents table
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    claim_code VARCHAR(255) UNIQUE,
    certificate TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create agent_teams table for team associations
CREATE TABLE agent_teams (
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, team_id)
);

-- Create agent_metrics table for storing performance data
CREATE TABLE agent_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    cpu_usage FLOAT,
    gpu_usage FLOAT,
    gpu_temp FLOAT,
    memory_usage FLOAT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_agents_claim_code ON agents(claim_code);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agent_teams_team_id ON agent_teams(team_id);
CREATE INDEX idx_agent_metrics_agent_id ON agent_metrics(agent_id);
CREATE INDEX idx_agent_metrics_timestamp ON agent_metrics(timestamp); 