-- Create agents table
CREATE TABLE agents (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    version VARCHAR(50) NOT NULL,
    hardware JSONB NOT NULL,
    os_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by_id UUID NOT NULL REFERENCES users(id),
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    extra_parameters TEXT DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    api_key VARCHAR(64) UNIQUE,
    api_key_created_at TIMESTAMP WITH TIME ZONE,
    api_key_last_used TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    max_agents INTEGER NOT NULL DEFAULT 1
);

COMMENT ON COLUMN agents.owner_id IS 'The user who owns/manages this agent';
COMMENT ON COLUMN agents.extra_parameters IS 'Agent-specific hashcat parameters (e.g., -w 4 -O)';
COMMENT ON COLUMN agents.is_enabled IS 'Whether the agent is enabled for job assignments';
COMMENT ON COLUMN agents.max_agents IS 'Maximum number of concurrent agents allowed with this claim code';

-- Create agent_metrics table
CREATE TABLE agent_metrics (
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    cpu_usage FLOAT NOT NULL,
    gpu_utilization FLOAT NOT NULL,
    gpu_temp FLOAT NOT NULL,
    memory_usage FLOAT NOT NULL,
    gpu_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, timestamp)
);

-- Create agent_teams junction table
CREATE TABLE agent_teams (
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    PRIMARY KEY (agent_id, team_id)
);

-- Create agent_devices table
CREATE TABLE agent_devices (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(50) NOT NULL CHECK (device_type IN ('CPU', 'GPU')),
    brand VARCHAR(50),
    total_memory BIGINT,
    driver_version VARCHAR(100),
    cuda_version VARCHAR(50),
    opencl_version VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, device_id)
);

COMMENT ON TABLE agent_devices IS 'Stores information about compute devices available on each agent';

-- Create agent_scheduling table
CREATE TABLE agent_scheduling (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    preferred_start_time TIME,
    preferred_end_time TIME,
    max_simultaneous_jobs INTEGER NOT NULL DEFAULT 1,
    allowed_job_priorities VARCHAR(50)[] DEFAULT ARRAY['low', 'medium', 'high', 'critical'],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
);

COMMENT ON TABLE agent_scheduling IS 'Stores scheduling preferences and constraints for each agent';
COMMENT ON COLUMN agent_scheduling.preferred_start_time IS 'Preferred daily start time for running jobs (in agent local timezone)';
COMMENT ON COLUMN agent_scheduling.preferred_end_time IS 'Preferred daily end time for running jobs (in agent local timezone)';
COMMENT ON COLUMN agent_scheduling.allowed_job_priorities IS 'Array of job priority levels this agent is allowed to process';

-- Create indexes
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_created_by ON agents(created_by_id);
CREATE INDEX idx_agents_owner_id ON agents(owner_id);
CREATE INDEX idx_agents_last_heartbeat ON agents(last_heartbeat);
CREATE INDEX idx_agents_api_key ON agents(api_key);
CREATE INDEX idx_agents_is_enabled ON agents(is_enabled);
CREATE INDEX idx_agent_metrics_timestamp ON agent_metrics(timestamp);
CREATE INDEX idx_agent_devices_agent_id ON agent_devices(agent_id);
CREATE INDEX idx_agent_devices_is_active ON agent_devices(is_active);
CREATE INDEX idx_agent_scheduling_agent_id ON agent_scheduling(agent_id);
CREATE INDEX idx_agent_scheduling_enabled ON agent_scheduling(enabled);

-- Create triggers
CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_devices_updated_at
    BEFORE UPDATE ON agent_devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_scheduling_updated_at
    BEFORE UPDATE ON agent_scheduling
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Set owner_id to created_by_id for all agents as default
-- This ensures owner_id is populated immediately
CREATE OR REPLACE FUNCTION set_agent_owner()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.owner_id IS NULL THEN
        NEW.owner_id := NEW.created_by_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_agent_owner_trigger
    BEFORE INSERT ON agents
    FOR EACH ROW
    EXECUTE FUNCTION set_agent_owner();