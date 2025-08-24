-- Drop triggers
DROP TRIGGER IF EXISTS update_user_mfa_settings_updated_at ON user_mfa_settings;
DROP TRIGGER IF EXISTS update_auth_settings_updated_at ON auth_settings;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop indexes
DROP INDEX IF EXISTS idx_mfa_sessions_session_token;
DROP INDEX IF EXISTS idx_mfa_sessions_user_id;
DROP INDEX IF EXISTS idx_mfa_verification_codes_user_id;
DROP INDEX IF EXISTS idx_user_backup_codes_code;
DROP INDEX IF EXISTS idx_user_backup_codes_user_id;
DROP INDEX IF EXISTS idx_login_attempts_created_at;
DROP INDEX IF EXISTS idx_login_attempts_ip_address;
DROP INDEX IF EXISTS idx_user_sessions_session_token;
DROP INDEX IF EXISTS idx_user_sessions_user_id;
DROP INDEX IF EXISTS idx_security_audit_log_created_at;
DROP INDEX IF EXISTS idx_security_audit_log_event_type;
DROP INDEX IF EXISTS idx_security_audit_log_user_id;
DROP INDEX IF EXISTS idx_password_reset_tokens_token;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP INDEX IF EXISTS idx_vouchers_used_by;
DROP INDEX IF EXISTS idx_vouchers_created_by;
DROP INDEX IF EXISTS idx_vouchers_code;
DROP INDEX IF EXISTS idx_auth_tokens_expires_at;
DROP INDEX IF EXISTS idx_auth_tokens_refresh_token;
DROP INDEX IF EXISTS idx_auth_tokens_token;
DROP INDEX IF EXISTS idx_auth_tokens_user_id;
DROP INDEX IF EXISTS idx_user_teams_team_id;
DROP INDEX IF EXISTS idx_user_teams_user_id;
DROP INDEX IF EXISTS idx_teams_name;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

-- Drop MFA tables
DROP TABLE IF EXISTS mfa_sessions;
DROP TABLE IF EXISTS mfa_verification_codes;
DROP TABLE IF EXISTS user_backup_codes;
DROP TABLE IF EXISTS user_mfa_settings;

-- Drop auth related tables
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS security_audit_log;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS auth_settings;
DROP TABLE IF EXISTS vouchers;
DROP TABLE IF EXISTS auth_tokens;

-- Drop core tables
DROP TABLE IF EXISTS user_teams;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;

-- Drop functions
DROP FUNCTION IF EXISTS update_last_updated_column();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";