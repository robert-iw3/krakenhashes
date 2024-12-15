-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    version VARCHAR(50) NOT NULL,
    hardware JSONB NOT NULL,
    created_by_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    certificate TEXT,
    private_key TEXT
);

-- Create agent_metrics table
CREATE TABLE IF NOT EXISTS agent_metrics (
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    cpu_usage FLOAT NOT NULL,
    gpu_usage FLOAT NOT NULL,
    gpu_temp FLOAT NOT NULL,
    memory_usage FLOAT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, timestamp)
);

-- Create claim_vouchers table
CREATE TABLE IF NOT EXISTS claim_vouchers (
    code VARCHAR(50) PRIMARY KEY,
    created_by_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_continuous BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    used_at TIMESTAMP WITH TIME ZONE,
    used_by_id INTEGER REFERENCES users(id)
);

-- Create claim_voucher_usage table for tracking attempts
CREATE TABLE IF NOT EXISTS claim_voucher_usage (
    id SERIAL PRIMARY KEY,
    voucher_code VARCHAR(50) NOT NULL REFERENCES claim_vouchers(code),
    attempted_by_id INTEGER NOT NULL REFERENCES users(id),
    attempted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN NOT NULL DEFAULT false,
    ip_address VARCHAR(45), -- IPv6 addresses can be up to 45 characters
    user_agent TEXT,
    error_message TEXT
);

-- Create agent_teams junction table
CREATE TABLE IF NOT EXISTS agent_teams (
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    team_id INTEGER NOT NULL REFERENCES teams(id),
    PRIMARY KEY (agent_id, team_id)
);

-- Create indexes
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_created_by ON agents(created_by_id);
CREATE INDEX idx_agents_last_heartbeat ON agents(last_heartbeat);
CREATE INDEX idx_agents_certificate ON agents(certificate);
CREATE INDEX idx_claim_vouchers_active ON claim_vouchers(is_active);
CREATE INDEX idx_claim_vouchers_created_by ON claim_vouchers(created_by_id);
CREATE INDEX idx_claim_voucher_usage_voucher ON claim_voucher_usage(voucher_code);
CREATE INDEX idx_claim_voucher_usage_attempted_by ON claim_voucher_usage(attempted_by_id);
CREATE INDEX idx_agent_metrics_timestamp ON agent_metrics(timestamp);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 