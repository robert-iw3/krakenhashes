-- Add pot-file support for storing and managing cracked passwords

-- Staging table for pot-file entries
CREATE TABLE potfile_staging (
    id SERIAL PRIMARY KEY,
    password TEXT NOT NULL,
    hash_value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ
);

-- Index for efficient querying of unprocessed entries
CREATE INDEX idx_potfile_staging_processed ON potfile_staging(processed) WHERE processed = FALSE;
-- Index for processed_at to help with cleanup of old processed entries
CREATE INDEX idx_potfile_staging_processed_at ON potfile_staging(processed_at) WHERE processed = TRUE;

-- Add columns to existing tables
ALTER TABLE wordlists ADD COLUMN is_potfile BOOLEAN DEFAULT FALSE;
ALTER TABLE job_tasks ADD COLUMN potfile_entries_added INTEGER DEFAULT 0;

-- Add index for quick lookup of pot-file wordlist
CREATE INDEX idx_wordlists_is_potfile ON wordlists(is_potfile) WHERE is_potfile = TRUE;

-- System settings for pot-file configuration
INSERT INTO system_settings (key, value, description, data_type) VALUES
    ('potfile_wordlist_id', NULL, 'ID of the pot-file wordlist entry', 'integer'),
    ('potfile_preset_job_id', NULL, 'ID of the pot-file preset job', 'uuid'),
    ('potfile_enabled', 'true', 'Whether pot-file feature is enabled', 'boolean'),
    ('potfile_batch_interval', '60', 'Seconds between pot-file batch processing', 'integer'),
    ('potfile_max_batch_size', '1000', 'Maximum entries to process in one batch', 'integer');

-- Add comments for documentation
COMMENT ON TABLE potfile_staging IS 'Staging table for passwords to be added to the pot-file';
COMMENT ON COLUMN potfile_staging.password IS 'The cracked password plaintext';
COMMENT ON COLUMN potfile_staging.hash_value IS 'The hash value that was cracked';
COMMENT ON COLUMN potfile_staging.processed IS 'Whether this entry has been processed and added to pot-file';
COMMENT ON COLUMN potfile_staging.processed_at IS 'Timestamp when the entry was processed';

COMMENT ON COLUMN wordlists.is_potfile IS 'Flag indicating if this wordlist is the system pot-file';
COMMENT ON COLUMN job_tasks.potfile_entries_added IS 'Number of pot-file entries added during this task execution';