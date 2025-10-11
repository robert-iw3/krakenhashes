-- Add accurate keyspace tracking columns for hashcat progress[1] integration
-- This migration adds support for capturing true effective keyspace from hashcat
-- rather than calculating it via simple multiplication

-- Job-level: Track if effective_keyspace is from hashcat or calculated
ALTER TABLE job_executions
ADD COLUMN avg_rule_multiplier NUMERIC(20,10),  -- Actual effectiveness: effective/base/rules
ADD COLUMN is_accurate_keyspace BOOLEAN DEFAULT FALSE;  -- TRUE when set by hashcat progress[1]

-- Task-level: Track if effective_keyspace_start/end are actual or estimated
ALTER TABLE job_tasks
ADD COLUMN is_actual_keyspace BOOLEAN DEFAULT FALSE;  -- TRUE when set by hashcat progress[1]

-- Index for filtering accurate vs legacy jobs
CREATE INDEX idx_job_executions_accurate ON job_executions(is_accurate_keyspace);

-- Documentation
COMMENT ON COLUMN job_executions.is_accurate_keyspace IS 'TRUE if effective_keyspace was set from hashcat progress[1], FALSE if calculated by multiplication';
COMMENT ON COLUMN job_executions.avg_rule_multiplier IS 'Actual multiplier: effective_keyspace / base_keyspace / rule_count. Used for estimating future tasks.';
COMMENT ON COLUMN job_tasks.is_actual_keyspace IS 'TRUE if effective_keyspace_start/end set from hashcat progress[1], FALSE if estimated';

-- Set defaults for existing data (mark as estimates/calculated)
UPDATE job_executions SET is_accurate_keyspace = false WHERE is_accurate_keyspace IS NULL;
UPDATE job_tasks SET is_actual_keyspace = false WHERE is_actual_keyspace IS NULL;
