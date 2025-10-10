-- Add potfile exclusion flag to clients table
-- This allows users to exclude specific clients from having their cracked passwords added to the potfile
-- Useful for clients with strict data retention requirements

ALTER TABLE clients ADD COLUMN exclude_from_potfile BOOLEAN DEFAULT FALSE NOT NULL;

-- Add index for efficient filtering of excluded clients
CREATE INDEX idx_clients_exclude_from_potfile ON clients(exclude_from_potfile) WHERE exclude_from_potfile = TRUE;

-- Add comment for documentation
COMMENT ON COLUMN clients.exclude_from_potfile IS 'Flag to exclude cracked passwords from this client''s hashlists from being added to the potfile. Useful for clients with strict data retention policies.';
