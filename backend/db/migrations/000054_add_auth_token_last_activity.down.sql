-- Drop index first
DROP INDEX IF EXISTS idx_auth_tokens_last_activity;

-- Remove last_activity column from auth_tokens table
ALTER TABLE auth_tokens DROP COLUMN IF EXISTS last_activity;