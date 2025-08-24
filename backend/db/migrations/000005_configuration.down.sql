-- Drop triggers
DROP TRIGGER IF EXISTS update_system_settings_updated_at ON system_settings;
DROP TRIGGER IF EXISTS update_client_settings_updated_at ON client_settings;
DROP TRIGGER IF EXISTS update_email_templates_updated_at ON email_templates;

-- Drop indexes
DROP INDEX IF EXISTS idx_email_templates_is_active;
DROP INDEX IF EXISTS idx_email_templates_template_type;

-- Drop tables
DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS client_settings;
DROP TABLE IF EXISTS email_templates;