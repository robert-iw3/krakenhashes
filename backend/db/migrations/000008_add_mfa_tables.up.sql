-- Create table for pending MFA setup
CREATE TABLE pending_mfa_setup (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(20) NOT NULL CHECK (method IN ('email', 'authenticator')),
    secret TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create table for email MFA codes
CREATE TABLE email_mfa_codes (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(6) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_pending_mfa_created_at ON pending_mfa_setup(created_at);
CREATE INDEX idx_email_mfa_expires_at ON email_mfa_codes(expires_at);

-- Add trigger for cleanup of expired entries
CREATE OR REPLACE FUNCTION cleanup_expired_mfa_entries()
RETURNS trigger AS $$
BEGIN
    -- Clean up expired pending setups (older than 15 minutes)
    DELETE FROM pending_mfa_setup 
    WHERE created_at < NOW() - INTERVAL '15 minutes';
    
    -- Clean up expired email codes
    DELETE FROM email_mfa_codes 
    WHERE expires_at < NOW();
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_cleanup_expired_mfa
    AFTER INSERT ON email_mfa_codes
    EXECUTE FUNCTION cleanup_expired_mfa_entries(); 