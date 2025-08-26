-- Remove dispatched_keyspace field
ALTER TABLE job_executions 
DROP COLUMN dispatched_keyspace;