-- Add dispatched_keyspace field to track total keyspace assigned to tasks
ALTER TABLE job_executions 
ADD COLUMN dispatched_keyspace BIGINT DEFAULT 0;

-- Update existing records to calculate dispatched_keyspace from existing tasks
UPDATE job_executions je
SET dispatched_keyspace = (
    SELECT COALESCE(SUM(
        CASE 
            WHEN jt.status IN ('assigned', 'running', 'completed', 'error') 
            THEN (jt.keyspace_end - jt.keyspace_start)
            ELSE 0
        END
    ), 0)
    FROM job_tasks jt
    WHERE jt.job_execution_id = je.id
)
WHERE je.status IN ('running', 'paused', 'completed');

-- Add comment explaining the field
COMMENT ON COLUMN job_executions.dispatched_keyspace IS 'Total keyspace that has been dispatched to tasks (assigned, running, completed, or error status)';