-- Drop triggers first
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_agents_status;
DROP INDEX IF EXISTS idx_agents_created_by;
DROP INDEX IF EXISTS idx_agents_last_heartbeat;
DROP INDEX IF EXISTS idx_claim_vouchers_active;
DROP INDEX IF EXISTS idx_claim_vouchers_created_by;
DROP INDEX IF EXISTS idx_claim_voucher_usage_voucher;
DROP INDEX IF EXISTS idx_claim_voucher_usage_attempted_by;
DROP INDEX IF EXISTS idx_agent_metrics_timestamp;

-- Drop tables in correct order to handle foreign key constraints
DROP TABLE IF EXISTS agent_teams;
DROP TABLE IF EXISTS agent_metrics;
DROP TABLE IF EXISTS claim_voucher_usage;
DROP TABLE IF EXISTS claim_vouchers;
DROP TABLE IF EXISTS agents; 