-- Drop indexes first
DROP INDEX IF EXISTS idx_agent_performance_metrics_device_lookup;
DROP INDEX IF EXISTS idx_agent_performance_metrics_task;

-- Remove device tracking columns from agent_performance_metrics table
ALTER TABLE agent_performance_metrics 
DROP COLUMN IF EXISTS device_id,
DROP COLUMN IF EXISTS device_name,
DROP COLUMN IF EXISTS task_id,
DROP COLUMN IF EXISTS attack_mode;