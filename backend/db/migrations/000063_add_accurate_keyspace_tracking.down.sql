-- Rollback accurate keyspace tracking

DROP INDEX IF EXISTS idx_job_executions_accurate;
ALTER TABLE job_tasks DROP COLUMN IF EXISTS is_actual_keyspace;
ALTER TABLE job_executions DROP COLUMN IF EXISTS avg_rule_multiplier;
ALTER TABLE job_executions DROP COLUMN IF EXISTS is_accurate_keyspace;
