-- Create email_templates table
CREATE TABLE email_templates (
    id SERIAL PRIMARY KEY,
    template_type VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    html_content TEXT NOT NULL,
    text_content TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE email_templates IS 'Stores email templates for various system notifications';

-- Create client_settings table
CREATE TABLE client_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE client_settings IS 'Stores global default settings related to clients';

-- Create system_settings table
CREATE TABLE system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    description TEXT,
    data_type VARCHAR(50) NOT NULL DEFAULT 'string',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_settings IS 'Stores global system-wide settings';
COMMENT ON COLUMN system_settings.data_type IS 'Data type of the value: string, integer, boolean, float';

-- Create indexes
CREATE INDEX idx_email_templates_template_type ON email_templates(template_type);
CREATE INDEX idx_email_templates_is_active ON email_templates(is_active);

-- Create triggers
CREATE TRIGGER update_email_templates_updated_at
    BEFORE UPDATE ON email_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_client_settings_updated_at
    BEFORE UPDATE ON client_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();