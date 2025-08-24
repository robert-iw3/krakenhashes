-- Drop triggers
DROP TRIGGER IF EXISTS set_agent_owner_trigger ON agents;
DROP TRIGGER IF EXISTS update_agent_scheduling_updated_at ON agent_scheduling;
DROP TRIGGER IF EXISTS update_agent_devices_updated_at ON agent_devices;
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;

-- Drop function
DROP FUNCTION IF EXISTS set_agent_owner();

-- Drop indexes
DROP INDEX IF EXISTS idx_agent_scheduling_enabled;
DROP INDEX IF EXISTS idx_agent_scheduling_agent_id;
DROP INDEX IF EXISTS idx_agent_devices_is_active;
DROP INDEX IF EXISTS idx_agent_devices_agent_id;
DROP INDEX IF EXISTS idx_agent_metrics_timestamp;
DROP INDEX IF EXISTS idx_agents_is_enabled;
DROP INDEX IF EXISTS idx_agents_api_key;
DROP INDEX IF EXISTS idx_agents_last_heartbeat;
DROP INDEX IF EXISTS idx_agents_owner_id;
DROP INDEX IF EXISTS idx_agents_created_by;
DROP INDEX IF EXISTS idx_agents_status;

-- Drop tables
DROP TABLE IF EXISTS agent_scheduling;
DROP TABLE IF EXISTS agent_devices;
DROP TABLE IF EXISTS agent_teams;
DROP TABLE IF EXISTS agent_metrics;
DROP TABLE IF EXISTS agents;