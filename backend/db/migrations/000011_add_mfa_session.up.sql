CREATE TABLE mfa_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mfa_sessions_user_id ON mfa_sessions(user_id);
CREATE INDEX idx_mfa_sessions_session_token ON mfa_sessions(session_token);
CREATE INDEX idx_mfa_sessions_expires_at ON mfa_sessions(expires_at);

-- Function to enforce max attempts limit
CREATE OR REPLACE FUNCTION enforce_mfa_max_attempts()
RETURNS trigger AS $$
DECLARE
    max_attempts INT;
BEGIN
    SELECT mfa_max_attempts INTO max_attempts FROM auth_settings LIMIT 1;
    IF NEW.attempts > max_attempts THEN
        RAISE EXCEPTION 'Maximum MFA attempts exceeded (limit: %)', max_attempts;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to enforce max attempts limit
CREATE TRIGGER enforce_mfa_max_attempts_trigger
    BEFORE INSERT OR UPDATE ON mfa_sessions
    FOR EACH ROW
    EXECUTE FUNCTION enforce_mfa_max_attempts();

-- Function to clean up expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_mfa_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM mfa_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Function for cleanup trigger
CREATE OR REPLACE FUNCTION trigger_cleanup_expired_mfa_sessions()
RETURNS trigger AS $$
BEGIN
    PERFORM cleanup_expired_mfa_sessions();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically clean up expired sessions
CREATE TRIGGER cleanup_expired_mfa_sessions_trigger
    AFTER INSERT ON mfa_sessions
    EXECUTE FUNCTION trigger_cleanup_expired_mfa_sessions();

COMMENT ON TABLE mfa_sessions IS 'Tracks MFA verification sessions during login';
COMMENT ON COLUMN mfa_sessions.session_token IS 'Temporary token used to link MFA verification to login attempt';
COMMENT ON COLUMN mfa_sessions.expires_at IS 'When this MFA session expires';
COMMENT ON COLUMN mfa_sessions.attempts IS 'Number of failed verification attempts'; 