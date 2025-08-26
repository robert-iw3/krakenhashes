-- Add last_activity column to auth_tokens table for tracking idle timeout
ALTER TABLE auth_tokens 
ADD COLUMN last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Create index for efficient queries on last_activity
CREATE INDEX idx_auth_tokens_last_activity ON auth_tokens(last_activity);

-- Update existing tokens to have current timestamp as last_activity
UPDATE auth_tokens SET last_activity = CURRENT_TIMESTAMP WHERE last_activity IS NULL;