-- Rollback agent scheduling system

-- Drop trigger
DROP TRIGGER IF EXISTS update_agent_schedules_updated_at ON agent_schedules;

-- Remove system setting
DELETE FROM system_settings WHERE key = 'agent_scheduling_enabled';

-- Remove scheduling columns from agents table
ALTER TABLE agents DROP COLUMN IF EXISTS schedule_timezone;
ALTER TABLE agents DROP COLUMN IF EXISTS scheduling_enabled;

-- Drop agent_schedules table
DROP TABLE IF EXISTS agent_schedules;