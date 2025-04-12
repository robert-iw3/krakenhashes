-- Create the clients table
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    contact_info TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data_retention_months INT DEFAULT NULL -- NULL means use system default, 0 means keep forever
);

CREATE INDEX idx_clients_name ON clients(name);

COMMENT ON TABLE clients IS 'Stores information about clients for whom hashlists are processed';

-- Create the hash_types table
CREATE TABLE hash_types (
    id INT PRIMARY KEY, -- Corresponds to hashcat mode number
    name VARCHAR(255) NOT NULL,
    description TEXT,
    example TEXT,
    needs_processing BOOLEAN NOT NULL DEFAULT FALSE,
    processing_logic JSONB, -- Store processing rules as JSON
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE hash_types IS 'Stores information about supported hash types, keyed by hashcat mode ID';

-- Create the hashlists table
CREATE TABLE hashlists (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL, -- Allow client deletion without deleting hashlists
    hash_type_id INT NOT NULL REFERENCES hash_types(id),
    file_path VARCHAR(1024),
    total_hashes INT NOT NULL DEFAULT 0,
    cracked_hashes INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('uploading', 'processing', 'ready', 'error')),
    error_message TEXT
);

CREATE INDEX idx_hashlists_user_id ON hashlists(user_id);
CREATE INDEX idx_hashlists_client_id ON hashlists(client_id);
CREATE INDEX idx_hashlists_hash_type_id ON hashlists(hash_type_id);
CREATE INDEX idx_hashlists_status ON hashlists(status);

COMMENT ON TABLE hashlists IS 'Stores metadata about uploaded hash lists';

-- Create the hashes table
CREATE TABLE hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hash_value TEXT NOT NULL, -- Removed UNIQUE constraint
    original_hash TEXT, -- Store the original hash if processing occurred
    username TEXT, -- Added username field, nullable
    hash_type_id INT NOT NULL REFERENCES hash_types(id),
    is_cracked BOOLEAN NOT NULL DEFAULT FALSE,
    password TEXT,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add index separately as UNIQUE constraint was removed
CREATE INDEX idx_hashes_hash_value ON hashes(hash_value);

COMMENT ON TABLE hashes IS 'Stores individual hash entries; hash_value is indexed via idx_hashes_hash_value';

-- Create the hashlist_hashes junction table
CREATE TABLE hashlist_hashes (
    hashlist_id BIGINT NOT NULL REFERENCES hashlists(id) ON DELETE CASCADE,
    hash_id UUID NOT NULL REFERENCES hashes(id) ON DELETE CASCADE,
    PRIMARY KEY (hashlist_id, hash_id)
);

CREATE INDEX idx_hashlist_hashes_hashlist_id ON hashlist_hashes(hashlist_id);
CREATE INDEX idx_hashlist_hashes_hash_id ON hashlist_hashes(hash_id);

COMMENT ON TABLE hashlist_hashes IS 'Junction table for the many-to-many relationship between hashlists and hashes';

-- Trigger function to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to clients table
CREATE TRIGGER update_clients_updated_at
BEFORE UPDATE ON clients
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to hashlists table
CREATE TRIGGER update_hashlists_updated_at
BEFORE UPDATE ON hashlists
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to hashes table
CREATE TRIGGER update_hashes_last_updated
BEFORE UPDATE ON hashes
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column(); 