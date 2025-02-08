-- Drop indexes
DROP INDEX IF EXISTS idx_users_mfa_enabled;
DROP INDEX IF EXISTS idx_users_account_locked;
DROP INDEX IF EXISTS idx_users_account_enabled;
DROP INDEX IF EXISTS idx_login_attempts_user_id;
DROP INDEX IF EXISTS idx_login_attempts_attempted_at;
DROP INDEX IF EXISTS idx_login_attempts_notified;
DROP INDEX IF EXISTS idx_active_sessions_user_id;
DROP INDEX IF EXISTS idx_active_sessions_last_active;
DROP INDEX IF EXISTS idx_tokens_token;
DROP INDEX IF EXISTS idx_tokens_user_id;
DROP INDEX IF EXISTS idx_tokens_revoked;

-- Drop tables
DROP TABLE IF EXISTS active_sessions;
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS auth_settings;
DROP TABLE IF EXISTS tokens;

-- Remove columns from users table
ALTER TABLE users
    DROP COLUMN IF EXISTS mfa_enabled,
    DROP COLUMN IF EXISTS mfa_type,
    DROP COLUMN IF EXISTS mfa_secret,
    DROP COLUMN IF EXISTS backup_codes,
    DROP COLUMN IF EXISTS last_password_change,
    DROP COLUMN IF EXISTS failed_login_attempts,
    DROP COLUMN IF EXISTS last_failed_attempt,
    DROP COLUMN IF EXISTS account_locked,
    DROP COLUMN IF EXISTS account_locked_until,
    DROP COLUMN IF EXISTS account_enabled,
    DROP COLUMN IF EXISTS last_login,
    DROP COLUMN IF EXISTS disabled_reason,
    DROP COLUMN IF EXISTS disabled_at,
    DROP COLUMN IF EXISTS disabled_by; 