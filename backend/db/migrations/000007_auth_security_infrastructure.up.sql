-- Add new columns to users table
ALTER TABLE users 
    ADD COLUMN mfa_enabled BOOLEAN DEFAULT FALSE,
    ADD COLUMN mfa_type text[] DEFAULT ARRAY['email'] CHECK (
        -- Ensure array is not empty and email is always present
        array_length(mfa_type, 1) > 0 
        AND 'email' = ANY(mfa_type)
        -- Ensure only valid values
        AND mfa_type <@ ARRAY['email', 'authenticator', 'backup']::text[]
    ),
    ADD COLUMN mfa_secret TEXT,
    ADD COLUMN backup_codes TEXT[], -- Store hashed backup codes
    ADD COLUMN last_password_change TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN failed_login_attempts INT DEFAULT 0,
    ADD COLUMN last_failed_attempt TIMESTAMP WITH TIME ZONE,
    ADD COLUMN account_locked BOOLEAN DEFAULT FALSE,
    ADD COLUMN account_locked_until TIMESTAMP WITH TIME ZONE,
    ADD COLUMN account_enabled BOOLEAN DEFAULT TRUE,
    ADD COLUMN last_login TIMESTAMP WITH TIME ZONE,
    ADD COLUMN disabled_reason TEXT,
    ADD COLUMN disabled_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN disabled_by UUID REFERENCES users(id);

-- Create tokens table for JWT storage
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_reason TEXT
);

-- Create index for token lookups
CREATE INDEX idx_tokens_token ON tokens(token);
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_revoked ON tokens(revoked);

-- Create auth_settings table
CREATE TABLE auth_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    min_password_length INT DEFAULT 15,
    require_uppercase BOOLEAN DEFAULT TRUE,
    require_lowercase BOOLEAN DEFAULT TRUE,
    require_numbers BOOLEAN DEFAULT TRUE,
    require_special_chars BOOLEAN DEFAULT TRUE,
    max_failed_attempts INT DEFAULT 5,
    lockout_duration_minutes INT DEFAULT 60,
    require_mfa BOOLEAN DEFAULT FALSE,
    jwt_expiry_minutes INT DEFAULT 60,
    display_timezone VARCHAR(50) DEFAULT 'UTC',
    notification_aggregation_minutes INT DEFAULT 60
);

-- Create login_attempts table
CREATE TABLE login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    username VARCHAR(255), -- Store even for non-existent users
    ip_address INET NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    attempted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    notified BOOLEAN DEFAULT FALSE
);

-- Create active_sessions table
CREATE TABLE active_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_users_mfa_enabled ON users(mfa_enabled);
CREATE INDEX idx_users_account_locked ON users(account_locked);
CREATE INDEX idx_users_account_enabled ON users(account_enabled);
CREATE INDEX idx_login_attempts_user_id ON login_attempts(user_id);
CREATE INDEX idx_login_attempts_attempted_at ON login_attempts(attempted_at);
CREATE INDEX idx_login_attempts_notified ON login_attempts(notified);
CREATE INDEX idx_active_sessions_user_id ON active_sessions(user_id);
CREATE INDEX idx_active_sessions_last_active ON active_sessions(last_active_at);

-- Insert default auth settings
INSERT INTO auth_settings (
    min_password_length,
    require_uppercase,
    require_lowercase,
    require_numbers,
    require_special_chars,
    max_failed_attempts,
    lockout_duration_minutes,
    require_mfa,
    jwt_expiry_minutes,
    display_timezone,
    notification_aggregation_minutes
) VALUES (
    15, -- min_password_length
    TRUE, -- require_uppercase
    TRUE, -- require_lowercase
    TRUE, -- require_numbers
    TRUE, -- require_special_chars
    5, -- max_failed_attempts
    60, -- lockout_duration_minutes
    FALSE, -- require_mfa
    60, -- jwt_expiry_minutes
    'UTC', -- display_timezone
    60 -- notification_aggregation_minutes
); 