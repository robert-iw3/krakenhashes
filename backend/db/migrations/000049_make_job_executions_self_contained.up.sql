-- Make job_executions self-contained by adding all configuration fields
-- This allows custom jobs to be created without preset_jobs

-- Add configuration fields to job_executions
ALTER TABLE job_executions
    ADD COLUMN name VARCHAR(255),
    ADD COLUMN wordlist_ids JSONB,
    ADD COLUMN rule_ids JSONB,
    ADD COLUMN mask VARCHAR(255),
    ADD COLUMN binary_version_id INT REFERENCES binary_versions(id),
    ADD COLUMN chunk_size_seconds INT DEFAULT 900,
    ADD COLUMN status_updates_enabled BOOLEAN DEFAULT true,
    ADD COLUMN is_small_job BOOLEAN DEFAULT false,
    ADD COLUMN allow_high_priority_override BOOLEAN DEFAULT false,
    ADD COLUMN additional_args TEXT,
    ADD COLUMN hash_type INT;

-- Make preset_job_id nullable for custom jobs
ALTER TABLE job_executions
    ALTER COLUMN preset_job_id DROP NOT NULL;

-- Add comment for clarity
COMMENT ON COLUMN job_executions.preset_job_id IS 'Reference to preset template if job was created from preset, NULL for custom jobs';
COMMENT ON COLUMN job_executions.name IS 'Job name - copied from preset or provided for custom jobs';
COMMENT ON COLUMN job_executions.wordlist_ids IS 'Array of wordlist IDs stored as JSONB';
COMMENT ON COLUMN job_executions.rule_ids IS 'Array of rule IDs stored as JSONB';
COMMENT ON COLUMN job_executions.is_small_job IS 'If true, process as single chunk without splitting';
COMMENT ON COLUMN job_executions.allow_high_priority_override IS 'If true, allows higher priority jobs to interrupt';

-- Create index on preset_job_id for jobs created from presets
CREATE INDEX idx_job_executions_preset_job ON job_executions(preset_job_id) WHERE preset_job_id IS NOT NULL;