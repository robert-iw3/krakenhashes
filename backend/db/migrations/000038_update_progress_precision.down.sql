-- Revert progress precision changes
ALTER TABLE job_executions 
ALTER COLUMN overall_progress_percent TYPE NUMERIC(5,2);

-- Revert task progress precision
ALTER TABLE job_tasks 
ALTER COLUMN progress_percent TYPE NUMERIC(5,2);