-- Add effective keyspace columns to job_tasks table for accurate progress tracking
-- These columns are used for rule splitting where effective keyspace = base_keyspace * rules

ALTER TABLE job_tasks 
ADD COLUMN effective_keyspace_start BIGINT,
ADD COLUMN effective_keyspace_end BIGINT,
ADD COLUMN effective_keyspace_processed BIGINT DEFAULT 0;

-- Add comments to explain the columns
COMMENT ON COLUMN job_tasks.effective_keyspace_start IS 'Effective start position for this chunk (base_keyspace * rule_start_index)';
COMMENT ON COLUMN job_tasks.effective_keyspace_end IS 'Effective end position for this chunk (base_keyspace * rule_end_index)';
COMMENT ON COLUMN job_tasks.effective_keyspace_processed IS 'Actual effective progress (words × rules processed)';

-- Update existing rule-split tasks to have effective keyspace values
-- This assumes job_executions.base_keyspace is populated
UPDATE job_tasks t
SET 
    effective_keyspace_start = COALESCE(t.rule_start_index, 0) * COALESCE(je.base_keyspace, t.keyspace_end - t.keyspace_start),
    effective_keyspace_end = COALESCE(t.rule_end_index, 1) * COALESCE(je.base_keyspace, t.keyspace_end - t.keyspace_start),
    effective_keyspace_processed = 0
FROM job_executions je
WHERE t.job_execution_id = je.id
  AND t.is_rule_split_task = true;

-- For non-rule-split tasks, copy the regular keyspace values
UPDATE job_tasks
SET 
    effective_keyspace_start = keyspace_start,
    effective_keyspace_end = keyspace_end,
    effective_keyspace_processed = keyspace_processed
WHERE is_rule_split_task = false OR is_rule_split_task IS NULL;