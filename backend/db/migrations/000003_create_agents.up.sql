-- Check if required tables exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE EXCEPTION 'Table "users" does not exist. Please run previous migrations first.';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'teams') THEN
        RAISE EXCEPTION 'Table "teams" does not exist. Please run previous migrations first.';
    END IF;
END $$;

-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    version VARCHAR(50) NOT NULL,
    hardware JSONB NOT NULL,
    os_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    api_key VARCHAR(64) UNIQUE,
    api_key_created_at TIMESTAMP WITH TIME ZONE,
    api_key_last_used TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create agent_metrics table
CREATE TABLE IF NOT EXISTS agent_metrics (
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
CREATE TABLE IF NOT EXISTS agent_teams (
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    PRIMARY KEY (agent_id, team_id)
);

-- Create indexes if they don't exist and their tables exist
DO $$ 
BEGIN
    -- Indexes for agents table
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'agents') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agents_status') THEN
            CREATE INDEX idx_agents_status ON agents(status);
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agents_created_by') THEN
            CREATE INDEX idx_agents_created_by ON agents(created_by_id);
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agents_last_heartbeat') THEN
            CREATE INDEX idx_agents_last_heartbeat ON agents(last_heartbeat);
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agents_api_key') THEN
            CREATE INDEX idx_agents_api_key ON agents(api_key);
        END IF;
    END IF;

    -- Index for agent_metrics table
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'agent_metrics') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agent_metrics_timestamp') THEN
            CREATE INDEX idx_agent_metrics_timestamp ON agent_metrics(timestamp);
        END IF;
    END IF;
END $$;

-- Create or replace trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Drop trigger if exists and create it
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 

-- DEVELOPMENT ONLY: Insert test agent data for development purposes
INSERT INTO agents (
    id, 
    name, 
    status, 
    last_heartbeat, 
    version, 
    hardware, 
    os_info, 
    created_by_id, 
    created_at, 
    updated_at, 
    api_key, 
    api_key_created_at, 
    api_key_last_used, 
    last_error, 
    metadata
)
VALUES (
    1, 
    'zerker-fw16', 
    'inactive', 
    '0001-01-01 00:00:00+00', 
    '1.0.0', 
    '{"cpus":null,"gpus":null,"network_interfaces":null}', 
    NULL, 
    'f1a50a5d-ff43-4617-bc21-61c54b214dee', 
    '2025-03-02 14:41:23.092376+00', 
    '2025-03-02 14:41:44.180703+00', 
    '2e72b8a5fa274bb05838959fd8f67d18762a409d6e7896f49b87aac568f7398d', 
    '2025-03-02 14:41:23.092376+00', 
    '2025-03-02 14:41:25.192295+00', 
    NULL, 
    NULL
)
ON CONFLICT (id) DO UPDATE 
SET 
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    api_key = EXCLUDED.api_key,
    api_key_created_at = EXCLUDED.api_key_created_at;

-- DEVELOPMENT ONLY: Reset the sequence if needed
SELECT setval('agents_id_seq', (SELECT MAX(id) FROM agents), true); 