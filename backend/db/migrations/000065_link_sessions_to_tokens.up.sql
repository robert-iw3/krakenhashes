-- Add token_id column to active_sessions table to link sessions to JWT tokens
-- When a token is deleted, the session will automatically be deleted via CASCADE
ALTER TABLE active_sessions
    ADD COLUMN token_id UUID REFERENCES tokens(id) ON DELETE CASCADE;

-- Create index for faster lookups
CREATE INDEX idx_active_sessions_token_id ON active_sessions(token_id);

-- Note: Existing sessions without token_id will remain in the database
-- They will not be automatically cleaned up but will be orphaned
-- Consider running a cleanup query after deployment if needed
