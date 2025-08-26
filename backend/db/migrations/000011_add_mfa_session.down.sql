DROP TRIGGER IF EXISTS cleanup_expired_mfa_sessions_trigger ON mfa_sessions;
DROP TRIGGER IF EXISTS enforce_mfa_max_attempts_trigger ON mfa_sessions;
DROP FUNCTION IF EXISTS cleanup_expired_mfa_sessions();
DROP FUNCTION IF EXISTS trigger_cleanup_expired_mfa_sessions();
DROP FUNCTION IF EXISTS enforce_mfa_max_attempts();
DROP TABLE IF EXISTS mfa_sessions;