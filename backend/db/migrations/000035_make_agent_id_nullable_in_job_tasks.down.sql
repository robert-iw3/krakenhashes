-- Rollback: Make agent_id NOT NULL again in job_tasks table

-- Remove the unassigned tasks index
DROP INDEX IF EXISTS idx_job_tasks_unassigned;

-- Restore the original index
DROP INDEX IF EXISTS idx_job_tasks_agent_status;
CREATE INDEX idx_job_tasks_agent_status ON job_tasks(agent_id, status);

-- Delete any tasks that don't have an agent_id before making it NOT NULL
DELETE FROM job_tasks WHERE agent_id IS NULL;

-- Make agent_id NOT NULL again
ALTER TABLE job_tasks 
ALTER COLUMN agent_id SET NOT NULL;