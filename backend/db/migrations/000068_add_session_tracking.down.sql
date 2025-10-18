-- Remove session_started_at column from active_sessions table
ALTER TABLE active_sessions
DROP COLUMN IF EXISTS session_started_at;
