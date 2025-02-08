-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_cleanup_expired_mfa ON email_mfa_codes;

-- Drop function
DROP FUNCTION IF EXISTS cleanup_expired_mfa_entries();

-- Drop tables
DROP TABLE IF EXISTS email_mfa_codes;
DROP TABLE IF EXISTS pending_mfa_setup; 