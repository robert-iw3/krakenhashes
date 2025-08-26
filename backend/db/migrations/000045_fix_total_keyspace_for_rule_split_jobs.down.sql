-- Revert total_keyspace to base_keyspace for rule-split jobs
-- This is the inverse of the up migration
UPDATE job_executions 
SET 
    total_keyspace = base_keyspace,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    uses_rule_splitting = true 
    AND base_keyspace IS NOT NULL;