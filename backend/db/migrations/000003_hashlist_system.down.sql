-- Drop triggers
DROP TRIGGER IF EXISTS update_binary_versions_updated_at ON binary_versions;
DROP TRIGGER IF EXISTS update_rules_updated_at ON rules;
DROP TRIGGER IF EXISTS update_wordlists_updated_at ON wordlists;
DROP TRIGGER IF EXISTS update_hashes_last_updated ON hashes;
DROP TRIGGER IF EXISTS update_hashlists_updated_at ON hashlists;
DROP TRIGGER IF EXISTS update_clients_updated_at ON clients;

-- Drop indexes
DROP INDEX IF EXISTS idx_potfile_entries_source;
DROP INDEX IF EXISTS idx_potfile_entries_hash_type_id;
DROP INDEX IF EXISTS idx_potfile_entries_hash_value;
DROP INDEX IF EXISTS idx_binary_versions_created_by;
DROP INDEX IF EXISTS idx_binary_versions_is_system;
DROP INDEX IF EXISTS idx_binary_versions_is_default;
DROP INDEX IF EXISTS idx_binary_versions_version;
DROP INDEX IF EXISTS idx_rules_created_by;
DROP INDEX IF EXISTS idx_rules_is_default;
DROP INDEX IF EXISTS idx_rules_name;
DROP INDEX IF EXISTS idx_wordlists_created_by;
DROP INDEX IF EXISTS idx_wordlists_is_default;
DROP INDEX IF EXISTS idx_wordlists_name;
DROP INDEX IF EXISTS idx_hashlist_hashes_hash_id;
DROP INDEX IF EXISTS idx_hashlist_hashes_hashlist_id;
DROP INDEX IF EXISTS idx_hashes_is_cracked;
DROP INDEX IF EXISTS idx_hashes_hash_type_id;
DROP INDEX IF EXISTS idx_hashes_hash_value;
DROP INDEX IF EXISTS idx_hashlists_status;
DROP INDEX IF EXISTS idx_hashlists_hash_type_id;
DROP INDEX IF EXISTS idx_hashlists_client_id;
DROP INDEX IF EXISTS idx_hashlists_user_id;
DROP INDEX IF EXISTS idx_clients_name;

-- Drop tables
DROP TABLE IF EXISTS potfile_entries;
DROP TABLE IF EXISTS binary_versions;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS wordlists;
DROP TABLE IF EXISTS hashlist_hashes;
DROP TABLE IF EXISTS hashes;
DROP TABLE IF EXISTS hashlists;
DROP TABLE IF EXISTS hash_types;
DROP TABLE IF EXISTS clients;