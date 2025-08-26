-- Remove preferred_mfa_method column from users table
ALTER TABLE users
    DROP COLUMN IF EXISTS preferred_mfa_method; 