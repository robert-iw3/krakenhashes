-- Drop audit log table and its indexes
DROP TABLE IF EXISTS binary_version_audit_log;

-- Drop binary versions table and its indexes
DROP TABLE IF EXISTS binary_versions;

-- Drop the binary type enum
DROP TYPE IF EXISTS binary_type;

-- Drop the compression type enum
DROP TYPE IF EXISTS compression_type; 