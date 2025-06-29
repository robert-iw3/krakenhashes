-- Drop indexes
DROP INDEX IF EXISTS idx_agents_device_detection_status;

-- Remove device detection columns from agents table
ALTER TABLE agents DROP COLUMN IF EXISTS device_detection_status;
ALTER TABLE agents DROP COLUMN IF EXISTS device_detection_error;
ALTER TABLE agents DROP COLUMN IF EXISTS device_detection_at;

-- Drop agent_devices table
DROP TABLE IF EXISTS agent_devices;