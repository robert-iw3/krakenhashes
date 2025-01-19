-- Drop triggers
DROP TRIGGER IF EXISTS update_email_config_updated_at ON email_config;
DROP TRIGGER IF EXISTS update_email_templates_updated_at ON email_templates;

-- Drop tables
DROP TABLE IF EXISTS email_usage;
DROP TABLE IF EXISTS email_templates;
DROP TABLE IF EXISTS email_config;

-- Drop types
DROP TYPE IF EXISTS email_template_type;
DROP TYPE IF EXISTS email_provider_type; 