-- Remove chunk_number column and its index
DROP INDEX IF EXISTS idx_job_tasks_job_chunk;
ALTER TABLE job_tasks DROP COLUMN IF EXISTS chunk_number;