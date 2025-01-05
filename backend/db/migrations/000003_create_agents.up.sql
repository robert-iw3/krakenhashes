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