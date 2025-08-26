-- Remove effective keyspace columns from job_tasks table
ALTER TABLE job_tasks 
DROP COLUMN IF EXISTS effective_keyspace_start,
DROP COLUMN IF EXISTS effective_keyspace_end,
DROP COLUMN IF EXISTS effective_keyspace_processed;