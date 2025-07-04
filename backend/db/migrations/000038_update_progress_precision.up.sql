-- Update progress precision to show decimals for small progress values
ALTER TABLE job_executions 
ALTER COLUMN overall_progress_percent TYPE NUMERIC(6,3);

-- Update task progress precision as well
ALTER TABLE job_tasks 
ALTER COLUMN progress_percent TYPE NUMERIC(6,3);