-- Rollback: Remove chunk_actual_keyspace column

ALTER TABLE job_tasks DROP COLUMN IF EXISTS chunk_actual_keyspace;
