-- Add session management settings to auth_settings table
ALTER TABLE auth_settings
ADD COLUMN token_cleanup_interval_seconds INT DEFAULT 60,
ADD COLUMN max_concurrent_sessions INT DEFAULT 0,
ADD COLUMN session_absolute_timeout_hours INT DEFAULT 0;

-- Update the existing row with default values
UPDATE auth_settings
SET
    token_cleanup_interval_seconds = 60,
    max_concurrent_sessions = 0,
    session_absolute_timeout_hours = 0
WHERE token_cleanup_interval_seconds IS NULL;
