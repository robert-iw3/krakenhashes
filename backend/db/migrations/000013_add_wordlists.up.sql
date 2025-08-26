-- Create enum for wordlist types
CREATE TYPE wordlist_type AS ENUM ('general', 'specialized', 'targeted', 'custom');

-- Create enum for wordlist formats
CREATE TYPE wordlist_format AS ENUM ('plaintext', 'compressed');

-- Create table for wordlists
CREATE TABLE wordlists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    wordlist_type wordlist_type NOT NULL,
    format wordlist_format NOT NULL DEFAULT 'plaintext',
    file_name VARCHAR(255) NOT NULL,
    md5_hash VARCHAR(32) NOT NULL,
    file_size BIGINT NOT NULL,
    word_count BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id),
    last_verified_at TIMESTAMP WITH TIME ZONE,
    verification_status VARCHAR(50) DEFAULT 'pending'
);

-- Create indexes for quick lookups
CREATE INDEX idx_wordlists_name ON wordlists(name);
CREATE INDEX idx_wordlists_type ON wordlists(wordlist_type);
CREATE INDEX idx_wordlists_verification ON wordlists(verification_status);
CREATE INDEX idx_wordlists_md5 ON wordlists(md5_hash);

-- Create table for wordlist audit log
CREATE TABLE wordlist_audit_log (
    id SERIAL PRIMARY KEY,
    wordlist_id INTEGER NOT NULL REFERENCES wordlists(id),
    action VARCHAR(50) NOT NULL,
    performed_by UUID NOT NULL REFERENCES users(id),
    performed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);

-- Create index for audit log queries
CREATE INDEX idx_wordlist_audit_wordlist_id ON wordlist_audit_log(wordlist_id);
CREATE INDEX idx_wordlist_audit_performed_at ON wordlist_audit_log(performed_at);

-- Add comments for documentation
COMMENT ON TABLE wordlists IS 'Stores information about wordlists used for password cracking';
COMMENT ON TABLE wordlist_audit_log IS 'Tracks all changes and actions performed on wordlists';
COMMENT ON COLUMN wordlists.wordlist_type IS 'Type of wordlist: general, specialized, targeted, or custom';
COMMENT ON COLUMN wordlists.format IS 'Format of the wordlist: plaintext or compressed';
COMMENT ON COLUMN wordlists.md5_hash IS 'MD5 hash of the wordlist file for integrity verification';
COMMENT ON COLUMN wordlists.file_size IS 'Size of the wordlist file in bytes';
COMMENT ON COLUMN wordlists.word_count IS 'Number of words in the wordlist, if known';
COMMENT ON COLUMN wordlists.verification_status IS 'Current verification status: pending, verified, failed';
COMMENT ON COLUMN wordlists.last_verified_at IS 'Timestamp of last successful verification of wordlist integrity';

-- Create table for wordlist tags
CREATE TABLE wordlist_tags (
    id SERIAL PRIMARY KEY,
    wordlist_id INTEGER NOT NULL REFERENCES wordlists(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id)
);

-- Create unique constraint and index for wordlist tags
CREATE UNIQUE INDEX idx_wordlist_tags_unique ON wordlist_tags(wordlist_id, tag);
CREATE INDEX idx_wordlist_tags_tag ON wordlist_tags(tag);

-- Add comments for documentation
COMMENT ON TABLE wordlist_tags IS 'Stores tags associated with wordlists for better categorization and searching'; 