-- Remove session management settings from auth_settings table
ALTER TABLE auth_settings
DROP COLUMN IF EXISTS token_cleanup_interval_seconds,
DROP COLUMN IF EXISTS max_concurrent_sessions,
DROP COLUMN IF EXISTS session_absolute_timeout_hours;
