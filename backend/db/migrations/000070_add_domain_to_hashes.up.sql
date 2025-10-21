-- Add domain column to hashes table for storing extracted domain information
-- Supports formats like DOMAIN\username, username@domain, and NetNTLM domain fields

ALTER TABLE hashes
ADD COLUMN domain TEXT;

-- Add index on domain for efficient filtering and queries
CREATE INDEX idx_hashes_domain ON hashes(domain);

-- Add index on username for efficient lookups (if not already exists)
CREATE INDEX IF NOT EXISTS idx_hashes_username ON hashes(username);

COMMENT ON COLUMN hashes.domain IS 'Extracted domain from hash formats like DOMAIN\username, user@domain, or NetNTLM';
