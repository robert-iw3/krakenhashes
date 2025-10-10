-- Add potfile exclusion flag to hashlists table
-- This allows users to exclude specific hashlists from being added to the potfile
-- Useful for clients with strict data retention requirements

ALTER TABLE hashlists ADD COLUMN exclude_from_potfile BOOLEAN DEFAULT FALSE NOT NULL;

-- Add index for efficient filtering of excluded hashlists
CREATE INDEX idx_hashlists_exclude_from_potfile ON hashlists(exclude_from_potfile) WHERE exclude_from_potfile = TRUE;

-- Add comment for documentation
COMMENT ON COLUMN hashlists.exclude_from_potfile IS 'Flag to exclude cracked passwords from this hashlist from being added to the potfile. Useful for clients with strict data retention policies.';
