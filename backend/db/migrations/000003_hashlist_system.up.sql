-- Create clients table
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    contact_info TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data_retention_months INT DEFAULT NULL -- NULL means use system default, 0 means keep forever
);

COMMENT ON TABLE clients IS 'Stores information about clients for whom hashlists are processed';

-- Create hash_types table
CREATE TABLE hash_types (
    id INT PRIMARY KEY, -- Corresponds to hashcat mode number
    name VARCHAR(255) NOT NULL,
    description TEXT,
    example TEXT,
    needs_processing BOOLEAN NOT NULL DEFAULT FALSE,
    processing_logic JSONB, -- Store processing rules as JSON
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    slow BOOLEAN NOT NULL DEFAULT FALSE -- Indicates if this is a slow hash algorithm
);

COMMENT ON TABLE hash_types IS 'Stores information about supported hash types, keyed by hashcat mode ID';

-- Create hashlists table
CREATE TABLE hashlists (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    hash_type_id INT NOT NULL REFERENCES hash_types(id),
    file_path VARCHAR(1024),
    total_hashes INT NOT NULL DEFAULT 0,
    cracked_hashes INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('uploading', 'processing', 'ready', 'error')),
    error_message TEXT
);

COMMENT ON TABLE hashlists IS 'Stores metadata about uploaded hash lists';

-- Create hashes table
CREATE TABLE hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hash_value TEXT NOT NULL,
    original_hash TEXT, -- Store the original hash if processing occurred
    username TEXT, -- Username field, nullable
    hash_type_id INT NOT NULL REFERENCES hash_types(id),
    is_cracked BOOLEAN NOT NULL DEFAULT FALSE,
    password TEXT,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE hashes IS 'Stores individual hash entries; hash_value is indexed via idx_hashes_hash_value';

-- Create hashlist_hashes junction table
CREATE TABLE hashlist_hashes (
    hashlist_id BIGINT NOT NULL REFERENCES hashlists(id) ON DELETE CASCADE,
    hash_id UUID NOT NULL REFERENCES hashes(id) ON DELETE CASCADE,
    PRIMARY KEY (hashlist_id, hash_id)
);

COMMENT ON TABLE hashlist_hashes IS 'Junction table for the many-to-many relationship between hashlists and hashes';

-- Create wordlists table
CREATE TABLE wordlists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    file_path VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL,
    line_count BIGINT DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id)
);

COMMENT ON TABLE wordlists IS 'Stores information about available wordlists for attacks';

-- Create rules table
CREATE TABLE rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    file_path VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL,
    line_count BIGINT DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id)
);

COMMENT ON TABLE rules IS 'Stores information about available rule files for attacks';

-- Create binary_versions table
CREATE TABLE binary_versions (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    file_path VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL,
    hash VARCHAR(128) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id)
);

COMMENT ON TABLE binary_versions IS 'Stores information about available hashcat binary versions';
COMMENT ON COLUMN binary_versions.is_system IS 'Indicates if this binary is the system-installed version (not managed by the application)';

-- Create potfile_entries table
CREATE TABLE potfile_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hash_value TEXT NOT NULL,
    plaintext TEXT NOT NULL,
    hash_type_id INT REFERENCES hash_types(id),
    source VARCHAR(50) NOT NULL DEFAULT 'import',
    imported_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT potfile_entries_source_check CHECK (source IN ('import', 'crack', 'manual'))
);

COMMENT ON TABLE potfile_entries IS 'Stores entries from imported potfiles and cracked hashes';
COMMENT ON COLUMN potfile_entries.source IS 'Origin of the entry: import (from potfile), crack (from job), manual (user added)';

-- Create indexes
CREATE INDEX idx_clients_name ON clients(name);
CREATE INDEX idx_hashlists_user_id ON hashlists(user_id);
CREATE INDEX idx_hashlists_client_id ON hashlists(client_id);
CREATE INDEX idx_hashlists_hash_type_id ON hashlists(hash_type_id);
CREATE INDEX idx_hashlists_status ON hashlists(status);
CREATE INDEX idx_hashes_hash_value ON hashes(hash_value);
CREATE INDEX idx_hashes_hash_type_id ON hashes(hash_type_id);
CREATE INDEX idx_hashes_is_cracked ON hashes(is_cracked);
CREATE INDEX idx_hashlist_hashes_hashlist_id ON hashlist_hashes(hashlist_id);
CREATE INDEX idx_hashlist_hashes_hash_id ON hashlist_hashes(hash_id);
CREATE INDEX idx_wordlists_name ON wordlists(name);
CREATE INDEX idx_wordlists_is_default ON wordlists(is_default);
CREATE INDEX idx_wordlists_created_by ON wordlists(created_by);
CREATE INDEX idx_rules_name ON rules(name);
CREATE INDEX idx_rules_is_default ON rules(is_default);
CREATE INDEX idx_rules_created_by ON rules(created_by);
CREATE INDEX idx_binary_versions_version ON binary_versions(version);
CREATE INDEX idx_binary_versions_is_default ON binary_versions(is_default);
CREATE INDEX idx_binary_versions_is_system ON binary_versions(is_system);
CREATE INDEX idx_binary_versions_created_by ON binary_versions(created_by);
CREATE INDEX idx_potfile_entries_hash_value ON potfile_entries(hash_value);
CREATE INDEX idx_potfile_entries_hash_type_id ON potfile_entries(hash_type_id);
CREATE INDEX idx_potfile_entries_source ON potfile_entries(source);

-- Create triggers
CREATE TRIGGER update_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hashlists_updated_at
    BEFORE UPDATE ON hashlists
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Use the correct function for hashes.last_updated
CREATE TRIGGER update_hashes_last_updated
    BEFORE UPDATE ON hashes
    FOR EACH ROW
    EXECUTE FUNCTION update_last_updated_column();

CREATE TRIGGER update_wordlists_updated_at
    BEFORE UPDATE ON wordlists
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_binary_versions_updated_at
    BEFORE UPDATE ON binary_versions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();