-- Add last_activity column to tokens table
ALTER TABLE tokens
ADD COLUMN last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Create index for efficient queries on last_activity
CREATE INDEX idx_tokens_last_activity ON tokens(last_activity);

-- Update existing tokens to have current timestamp as last_activity
UPDATE tokens SET last_activity = CURRENT_TIMESTAMP WHERE last_activity IS NULL;

-- Drop the unused auth_tokens table
DROP TABLE IF EXISTS auth_tokens;
