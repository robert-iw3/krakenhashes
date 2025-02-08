-- Remove MFA-related columns from auth_settings
ALTER TABLE auth_settings
    DROP COLUMN allowed_mfa_methods,
    DROP COLUMN email_code_validity_minutes,
    DROP COLUMN backup_codes_count;

-- Remove MFA settings columns from auth_settings table
ALTER TABLE auth_settings
    DROP COLUMN mfa_code_cooldown_minutes,
    DROP COLUMN mfa_code_expiry_minutes,
    DROP COLUMN mfa_max_attempts; 