-- Remove the unique index for default binaries
DROP INDEX IF EXISTS idx_binary_versions_one_default_per_type;

-- Remove the is_default column
ALTER TABLE binary_versions 
DROP COLUMN IF EXISTS is_default;