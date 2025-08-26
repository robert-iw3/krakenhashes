-- Add agent scheduling system tables and columns

-- Create agent_schedules table to store daily schedules
CREATE TABLE agent_schedules (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,  -- Stored in UTC
    end_time TIME NOT NULL,    -- Stored in UTC
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',  -- Store the original timezone for reference
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_agent_day UNIQUE (agent_id, day_of_week),
    CONSTRAINT valid_time_range CHECK (
        -- Allow overnight schedules (e.g., 22:00 - 02:00)
        end_time != start_time
    )
);

-- Create indexes for performance
CREATE INDEX idx_agent_schedules_agent_id ON agent_schedules(agent_id);
CREATE INDEX idx_agent_schedules_day_active ON agent_schedules(day_of_week, is_active);

-- Add comments to document the purpose
COMMENT ON TABLE agent_schedules IS 'Stores daily scheduling information for agents to control when they can accept jobs';
COMMENT ON COLUMN agent_schedules.day_of_week IS 'Day of week: 0=Sunday, 1=Monday, ..., 6=Saturday';
COMMENT ON COLUMN agent_schedules.start_time IS 'Start time in UTC';
COMMENT ON COLUMN agent_schedules.end_time IS 'End time in UTC';
COMMENT ON COLUMN agent_schedules.timezone IS 'Original timezone for display purposes';

-- Add scheduling fields to agents table
ALTER TABLE agents ADD COLUMN scheduling_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE agents ADD COLUMN schedule_timezone VARCHAR(50) DEFAULT 'UTC';

COMMENT ON COLUMN agents.scheduling_enabled IS 'Whether scheduling is enabled for this agent';
COMMENT ON COLUMN agents.schedule_timezone IS 'Default timezone for agent schedules';

-- Add system setting for global scheduling control
INSERT INTO system_settings (key, value, description, data_type) 
VALUES ('agent_scheduling_enabled', 'false', 'Enable agent scheduling system globally', 'boolean')
ON CONFLICT (key) DO NOTHING;

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_agent_schedules_updated_at
    BEFORE UPDATE ON agent_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();