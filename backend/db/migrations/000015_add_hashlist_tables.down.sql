-- Drop the triggers first
DROP TRIGGER IF EXISTS update_hashes_last_updated ON hashes;
DROP TRIGGER IF EXISTS update_hashlists_updated_at ON hashlists;
DROP TRIGGER IF EXISTS update_clients_updated_at ON clients;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop the junction table
DROP TABLE IF EXISTS hashlist_hashes;

-- Drop the main tables (in reverse order of dependency)
DROP TABLE IF EXISTS hashes;
DROP TABLE IF EXISTS hashlists;
DROP TABLE IF EXISTS hash_types;
DROP TABLE IF EXISTS clients;

COMMENT ON TABLE hashlist_hashes IS NULL;
COMMENT ON TABLE hashes IS NULL;
COMMENT ON TABLE hashlists IS NULL;
COMMENT ON TABLE hash_types IS NULL;
COMMENT ON TABLE clients IS NULL; 