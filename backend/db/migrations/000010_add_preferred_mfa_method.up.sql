-- Add preferred_mfa_method column to users table with enhanced constraints
ALTER TABLE users
    ADD COLUMN preferred_mfa_method VARCHAR(20) DEFAULT 'email' 
    CHECK (
        preferred_mfa_method IN ('email', 'authenticator')
        AND preferred_mfa_method != 'backup'
    ),
    ADD CONSTRAINT users_preferred_mfa_method_in_types 
    CHECK (preferred_mfa_method = ANY(mfa_type));

-- Add comment for clarity
COMMENT ON COLUMN users.preferred_mfa_method IS 'User''s preferred MFA method for authentication';

-- Set initial values based on existing mfa_type
UPDATE users 
SET preferred_mfa_method = 'email'
WHERE mfa_enabled = true; 