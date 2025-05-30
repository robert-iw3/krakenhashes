-- Create the system_settings table
CREATE TABLE system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    description TEXT,
    data_type VARCHAR(50) NOT NULL DEFAULT 'string',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_settings IS 'Stores global system-wide settings';
COMMENT ON COLUMN system_settings.data_type IS 'Data type of the value: string, integer, boolean, float';

-- Insert default max priority setting
INSERT INTO system_settings (key, value, description, data_type)
VALUES ('max_job_priority', '1000', 'Maximum priority value that can be assigned to jobs and preset jobs', 'integer');

-- Apply trigger to system_settings table
CREATE TRIGGER update_system_settings_updated_at
BEFORE UPDATE ON system_settings
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column(); 