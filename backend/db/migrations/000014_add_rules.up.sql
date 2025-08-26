-- Create enum for rule types
CREATE TYPE rule_type AS ENUM ('hashcat', 'john');

-- Create table for rules
CREATE TABLE rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type rule_type NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    md5_hash VARCHAR(32) NOT NULL,
    file_size BIGINT NOT NULL,
    rule_count INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id),
    last_verified_at TIMESTAMP WITH TIME ZONE,
    verification_status VARCHAR(50) DEFAULT 'pending',
    estimated_keyspace_multiplier FLOAT
);

-- Create indexes for quick lookups
CREATE INDEX idx_rules_name ON rules(name);
CREATE INDEX idx_rules_type ON rules(rule_type);
CREATE INDEX idx_rules_verification ON rules(verification_status);
CREATE INDEX idx_rules_md5 ON rules(md5_hash);

-- Create table for rule audit log
CREATE TABLE rule_audit_log (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES rules(id),
    action VARCHAR(50) NOT NULL,
    performed_by UUID NOT NULL REFERENCES users(id),
    performed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);

-- Create index for audit log queries
CREATE INDEX idx_rule_audit_rule_id ON rule_audit_log(rule_id);
CREATE INDEX idx_rule_audit_performed_at ON rule_audit_log(performed_at);

-- Add comments for documentation
COMMENT ON TABLE rules IS 'Stores information about rules used for password cracking';
COMMENT ON TABLE rule_audit_log IS 'Tracks all changes and actions performed on rules';
COMMENT ON COLUMN rules.rule_type IS 'Type of rule: hashcat or john';
COMMENT ON COLUMN rules.md5_hash IS 'MD5 hash of the rule file for integrity verification';
COMMENT ON COLUMN rules.file_size IS 'Size of the rule file in bytes';
COMMENT ON COLUMN rules.rule_count IS 'Number of rules in the file, if known';
COMMENT ON COLUMN rules.verification_status IS 'Current verification status: pending, verified, failed';
COMMENT ON COLUMN rules.last_verified_at IS 'Timestamp of last successful verification of rule integrity';
COMMENT ON COLUMN rules.estimated_keyspace_multiplier IS 'Estimated factor by which this rule multiplies the keyspace';

-- Create table for rule tags
CREATE TABLE rule_tags (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id)
);

-- Create unique constraint and index for rule tags
CREATE UNIQUE INDEX idx_rule_tags_unique ON rule_tags(rule_id, tag);
CREATE INDEX idx_rule_tags_tag ON rule_tags(tag);

-- Add comments for documentation
COMMENT ON TABLE rule_tags IS 'Stores tags associated with rules for better categorization and searching';

-- Create table for rule-wordlist compatibility
CREATE TABLE rule_wordlist_compatibility (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    wordlist_id INTEGER NOT NULL REFERENCES wordlists(id) ON DELETE CASCADE,
    compatibility_score FLOAT NOT NULL DEFAULT 1.0,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id)
);

-- Create unique constraint and indexes
CREATE UNIQUE INDEX idx_rule_wordlist_unique ON rule_wordlist_compatibility(rule_id, wordlist_id);
CREATE INDEX idx_rule_wordlist_rule ON rule_wordlist_compatibility(rule_id);
CREATE INDEX idx_rule_wordlist_wordlist ON rule_wordlist_compatibility(wordlist_id);

-- Add comments for documentation
COMMENT ON TABLE rule_wordlist_compatibility IS 'Stores compatibility information between rules and wordlists';
COMMENT ON COLUMN rule_wordlist_compatibility.compatibility_score IS 'Score from 0.0 to 1.0 indicating how well the rule works with the wordlist'; 