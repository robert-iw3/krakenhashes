-- Rollback: Remove Enhanced Chunking Support

-- Remove system settings for rule splitting
DELETE FROM system_settings WHERE key IN (
    'rule_split_enabled',
    'rule_split_threshold',
    'rule_split_min_rules',
    'rule_split_max_chunks',
    'rule_chunk_temp_dir'
);

-- Drop indexes from job_tasks table
DROP INDEX IF EXISTS idx_job_tasks_rule_split;

-- Remove columns from job_tasks table
ALTER TABLE job_tasks
DROP COLUMN IF EXISTS rule_start_index,
DROP COLUMN IF EXISTS rule_end_index,
DROP COLUMN IF EXISTS rule_chunk_path,
DROP COLUMN IF EXISTS is_rule_split_task,
DROP COLUMN IF EXISTS priority,
DROP COLUMN IF EXISTS attack_cmd;

-- Drop indexes from job_executions table
DROP INDEX IF EXISTS idx_job_executions_rule_splitting;

-- Remove columns from job_executions table
ALTER TABLE job_executions
DROP COLUMN IF EXISTS base_keyspace,
DROP COLUMN IF EXISTS effective_keyspace,
DROP COLUMN IF EXISTS multiplication_factor,
DROP COLUMN IF EXISTS uses_rule_splitting,
DROP COLUMN IF EXISTS rule_split_count;