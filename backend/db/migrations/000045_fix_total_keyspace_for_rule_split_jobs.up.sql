-- Fix total_keyspace for existing rule-split jobs
-- This ensures percentage calculations use the correct denominator
UPDATE job_executions 
SET 
    total_keyspace = effective_keyspace,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    uses_rule_splitting = true 
    AND effective_keyspace IS NOT NULL
    AND (total_keyspace IS NULL OR total_keyspace != effective_keyspace);