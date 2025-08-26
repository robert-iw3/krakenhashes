-- Create enum for binary types
CREATE TYPE binary_type AS ENUM ('hashcat', 'john');

-- Create enum for compression types
CREATE TYPE compression_type AS ENUM ('7z', 'zip', 'tar.gz', 'tar.xz');

-- Create table for binary versions
CREATE TABLE binary_versions (
    id SERIAL PRIMARY KEY,
    binary_type binary_type NOT NULL,
    compression_type compression_type NOT NULL,
    source_url TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    md5_hash VARCHAR(32) NOT NULL,
    file_size BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id),
    is_active BOOLEAN DEFAULT true,
    last_verified_at TIMESTAMP WITH TIME ZONE,
    verification_status VARCHAR(50) DEFAULT 'pending'
);

-- Create index for quick lookups
CREATE INDEX idx_binary_versions_type_active ON binary_versions(binary_type) WHERE is_active = true;
CREATE INDEX idx_binary_versions_verification ON binary_versions(verification_status);

-- Create table for binary version audit log
CREATE TABLE binary_version_audit_log (
    id SERIAL PRIMARY KEY,
    binary_version_id INTEGER NOT NULL REFERENCES binary_versions(id),
    action VARCHAR(50) NOT NULL,
    performed_by UUID NOT NULL REFERENCES users(id),
    performed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);

-- Create index for audit log queries
CREATE INDEX idx_binary_version_audit_binary_id ON binary_version_audit_log(binary_version_id);
CREATE INDEX idx_binary_version_audit_performed_at ON binary_version_audit_log(performed_at);

-- Add comments for documentation
COMMENT ON TABLE binary_versions IS 'Stores information about different versions of hash cracking binaries';
COMMENT ON TABLE binary_version_audit_log IS 'Tracks all changes and actions performed on binary versions';
COMMENT ON COLUMN binary_versions.verification_status IS 'Current verification status: pending, verified, failed';
COMMENT ON COLUMN binary_versions.last_verified_at IS 'Timestamp of last successful verification of binary integrity';
COMMENT ON COLUMN binary_versions.file_size IS 'Size of the binary file in bytes';
COMMENT ON COLUMN binary_versions.compression_type IS 'Type of compression used for the binary archive'; 