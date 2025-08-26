-- Drop tables in reverse order to avoid foreign key constraints
DROP TABLE IF EXISTS rule_wordlist_compatibility;
DROP TABLE IF EXISTS rule_tags;
DROP TABLE IF EXISTS rule_audit_log;
DROP TABLE IF EXISTS rules;

-- Drop enums
DROP TYPE IF EXISTS rule_type; 