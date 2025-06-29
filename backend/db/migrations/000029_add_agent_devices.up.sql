-- Create agent_devices table to track individual compute devices
CREATE TABLE agent_devices (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(50) NOT NULL, -- 'GPU' or 'CPU'
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, device_id)
);

-- Create index for faster lookups
CREATE INDEX idx_agent_devices_agent_id ON agent_devices(agent_id);
CREATE INDEX idx_agent_devices_enabled ON agent_devices(agent_id, enabled);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_agent_devices_updated_at
    BEFORE UPDATE ON agent_devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add device detection status to agents table
ALTER TABLE agents ADD COLUMN device_detection_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE agents ADD COLUMN device_detection_error TEXT;
ALTER TABLE agents ADD COLUMN device_detection_at TIMESTAMP WITH TIME ZONE;

-- Add index for device detection status
CREATE INDEX idx_agents_device_detection_status ON agents(device_detection_status);