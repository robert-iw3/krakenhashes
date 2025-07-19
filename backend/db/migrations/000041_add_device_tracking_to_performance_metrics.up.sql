-- Drop the unused agent_metrics table
DROP TABLE IF EXISTS agent_metrics;

-- Add device tracking columns to agent_performance_metrics table
ALTER TABLE agent_performance_metrics 
ADD COLUMN IF NOT EXISTS device_id INTEGER,
ADD COLUMN IF NOT EXISTS device_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS task_id UUID REFERENCES job_tasks(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS attack_mode INTEGER;

-- Create composite index for efficient querying
CREATE INDEX IF NOT EXISTS idx_agent_performance_metrics_device_lookup 
ON agent_performance_metrics(agent_id, device_id, metric_type, timestamp DESC);

-- Create index for task-based queries
CREATE INDEX IF NOT EXISTS idx_agent_performance_metrics_task 
ON agent_performance_metrics(task_id) WHERE task_id IS NOT NULL;

-- Add comment to clarify the purpose of these columns
COMMENT ON COLUMN agent_performance_metrics.device_id IS 'Device ID from hashcat (GPU/CPU identifier)';
COMMENT ON COLUMN agent_performance_metrics.device_name IS 'Human-readable device name (e.g., "AMD Radeon RX 7700S")';
COMMENT ON COLUMN agent_performance_metrics.task_id IS 'Link to job_tasks for correlating metrics with specific job executions';
COMMENT ON COLUMN agent_performance_metrics.attack_mode IS 'Hashcat attack mode for future analytics';