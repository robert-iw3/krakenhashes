-- Create enum for email provider types
CREATE TYPE email_provider_type AS ENUM ('mailgun', 'sendgrid', 'mailchimp', 'gmail');

-- Create enum for email template types
CREATE TYPE email_template_type AS ENUM ('security_event', 'job_completion', 'admin_error', 'mfa_code');

-- Create email configuration table
CREATE TABLE email_config (
    id SERIAL PRIMARY KEY,
    provider_type email_provider_type NOT NULL,
    api_key TEXT NOT NULL,
    additional_config JSONB,
    monthly_limit INTEGER,
    reset_date TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create email templates table
CREATE TABLE email_templates (
    id SERIAL PRIMARY KEY,
    template_type email_template_type NOT NULL,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_content TEXT NOT NULL,
    text_content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_modified_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Create email usage tracking table
CREATE TABLE email_usage (
    id SERIAL PRIMARY KEY,
    month_year DATE NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    last_reset TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_month_year UNIQUE (month_year)
);

-- Add indexes
CREATE INDEX idx_email_templates_type ON email_templates(template_type);
CREATE INDEX idx_email_usage_month_year ON email_usage(month_year);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_email_config_updated_at
    BEFORE UPDATE ON email_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_email_templates_updated_at
    BEFORE UPDATE ON email_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 