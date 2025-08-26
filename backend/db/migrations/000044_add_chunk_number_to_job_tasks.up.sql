-- Add chunk_number column to job_tasks table for better user tracking
-- Each job will have its own chunk numbering sequence (1, 2, 3...)
ALTER TABLE job_tasks ADD COLUMN chunk_number INTEGER;

-- Add index for efficient queries by job and chunk number
CREATE INDEX idx_job_tasks_job_chunk ON job_tasks(job_execution_id, chunk_number);

-- Update existing tasks to have chunk numbers based on creation order
WITH numbered_tasks AS (
    SELECT id, 
           ROW_NUMBER() OVER (PARTITION BY job_execution_id ORDER BY created_at) as chunk_num
    FROM job_tasks
)
UPDATE job_tasks 
SET chunk_number = numbered_tasks.chunk_num
FROM numbered_tasks
WHERE job_tasks.id = numbered_tasks.id;