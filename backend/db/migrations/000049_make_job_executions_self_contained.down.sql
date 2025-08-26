-- Revert job_executions to reference-based model

-- Drop index
DROP INDEX IF EXISTS idx_job_executions_preset_job;

-- Remove comments
COMMENT ON COLUMN job_executions.preset_job_id IS NULL;

-- Make preset_job_id required again
ALTER TABLE job_executions
    ALTER COLUMN preset_job_id SET NOT NULL;

-- Remove configuration fields
ALTER TABLE job_executions
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS wordlist_ids,
    DROP COLUMN IF EXISTS rule_ids,
    DROP COLUMN IF EXISTS mask,
    DROP COLUMN IF EXISTS binary_version_id,
    DROP COLUMN IF EXISTS chunk_size_seconds,
    DROP COLUMN IF EXISTS status_updates_enabled,
    DROP COLUMN IF EXISTS is_small_job,
    DROP COLUMN IF EXISTS allow_high_priority_override,
    DROP COLUMN IF EXISTS additional_args,
    DROP COLUMN IF EXISTS hash_type;