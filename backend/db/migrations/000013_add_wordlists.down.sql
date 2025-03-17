-- Drop tables in reverse order to avoid foreign key constraints
DROP TABLE IF EXISTS wordlist_tags;
DROP TABLE IF EXISTS wordlist_audit_log;
DROP TABLE IF EXISTS wordlists;

-- Drop enums
DROP TYPE IF EXISTS wordlist_format;
DROP TYPE IF EXISTS wordlist_type; 