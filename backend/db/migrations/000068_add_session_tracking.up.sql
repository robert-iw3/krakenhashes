-- Add session_started_at to track original session start time (persists across token refreshes)
ALTER TABLE active_sessions
ADD COLUMN session_started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Update existing sessions to use created_at as session_started_at
UPDATE active_sessions
SET session_started_at = created_at
WHERE session_started_at IS NULL;

-- Make session_started_at NOT NULL after setting values
ALTER TABLE active_sessions
ALTER COLUMN session_started_at SET NOT NULL;
