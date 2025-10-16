-- Remove the index
DROP INDEX IF EXISTS idx_active_sessions_token_id;

-- Remove the token_id column from active_sessions table
ALTER TABLE active_sessions
    DROP COLUMN IF EXISTS token_id;
