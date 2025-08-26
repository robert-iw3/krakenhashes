-- Add new MFA-related columns to auth_settings
ALTER TABLE auth_settings
    ADD COLUMN allowed_mfa_methods JSONB DEFAULT '["email", "authenticator"]'::jsonb,
    ADD COLUMN email_code_validity_minutes INT DEFAULT 5,
    ADD COLUMN backup_codes_count INT DEFAULT 8,
    ADD COLUMN mfa_code_cooldown_minutes INT DEFAULT 1,
    ADD COLUMN mfa_code_expiry_minutes INT DEFAULT 5,
    ADD COLUMN mfa_max_attempts INT DEFAULT 3;

-- Add comment for clarity
COMMENT ON TABLE auth_settings IS 'Stores global authentication and security settings';
COMMENT ON COLUMN auth_settings.mfa_code_cooldown_minutes IS 'Cooldown period in minutes between MFA code requests';
COMMENT ON COLUMN auth_settings.mfa_code_expiry_minutes IS 'Time in minutes before an MFA code expires';
COMMENT ON COLUMN auth_settings.mfa_max_attempts IS 'Maximum number of failed attempts before an MFA code is invalidated';

-- Update existing settings with default values
UPDATE auth_settings
SET 
    mfa_code_cooldown_minutes = 1,
    mfa_code_expiry_minutes = 5,
    mfa_max_attempts = 3; 