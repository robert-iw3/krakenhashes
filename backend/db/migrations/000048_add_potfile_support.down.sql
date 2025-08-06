-- Remove pot-file support

-- Remove system settings
DELETE FROM system_settings WHERE key IN (
    'potfile_wordlist_id',
    'potfile_preset_job_id', 
    'potfile_enabled',
    'potfile_batch_interval',
    'potfile_max_batch_size'
);

-- Drop indexes
DROP INDEX IF EXISTS idx_wordlists_is_potfile;
DROP INDEX IF EXISTS idx_potfile_staging_processed_at;
DROP INDEX IF EXISTS idx_potfile_staging_processed;

-- Remove columns from existing tables
ALTER TABLE job_tasks DROP COLUMN IF EXISTS potfile_entries_added;
ALTER TABLE wordlists DROP COLUMN IF EXISTS is_potfile;

-- Drop staging table
DROP TABLE IF EXISTS potfile_staging;