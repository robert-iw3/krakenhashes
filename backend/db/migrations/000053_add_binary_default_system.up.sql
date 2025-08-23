-- Add is_default column to binary_versions table
ALTER TABLE binary_versions 
ADD COLUMN is_default BOOLEAN DEFAULT FALSE;

-- Create a partial unique index to ensure only one default per binary_type
CREATE UNIQUE INDEX idx_binary_versions_one_default_per_type 
ON binary_versions (binary_type) 
WHERE is_default = TRUE AND is_active = TRUE;

-- Set initial defaults: the latest verified version for each binary_type becomes the default
UPDATE binary_versions bv1
SET is_default = TRUE
WHERE bv1.id = (
    SELECT id 
    FROM binary_versions bv2
    WHERE bv2.binary_type = bv1.binary_type 
    AND bv2.is_active = TRUE 
    AND bv2.verification_status = 'verified'
    ORDER BY bv2.created_at DESC
    LIMIT 1
);

-- Add comment for documentation
COMMENT ON COLUMN binary_versions.is_default IS 'Indicates if this is the default binary version for its type. Only one default allowed per binary_type when active.';