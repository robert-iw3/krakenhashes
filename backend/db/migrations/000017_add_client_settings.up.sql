-- Create the client_settings table
CREATE TABLE client_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE client_settings IS 'Stores global default settings related to clients';

-- Insert default retention setting (0 = keep forever)
INSERT INTO client_settings (key, value, description)
VALUES ('default_data_retention_months', '0', 'Default data retention period in months for clients without a specific setting. 0 means keep forever.');

-- Trigger function reference (assuming it exists from migration 15)
-- Apply trigger to client_settings table
CREATE TRIGGER update_client_settings_updated_at
BEFORE UPDATE ON client_settings
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column(); 