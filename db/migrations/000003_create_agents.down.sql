-- Drop triggers
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_agent_metrics_timestamp;
DROP INDEX IF EXISTS idx_agents_last_seen;
DROP INDEX IF EXISTS idx_agents_created_by;
DROP INDEX IF EXISTS idx_agents_status;

-- Drop tables
DROP TABLE IF EXISTS agent_teams;
DROP TABLE IF EXISTS agent_metrics;
DROP TABLE IF EXISTS agents; 