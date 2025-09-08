-- Add reconnect grace period setting for agent reconnection after server restart
INSERT INTO system_settings (key, value, description) 
VALUES (
    'reconnect_grace_period_minutes', 
    '5', 
    'Grace period in minutes for agents to reconnect after server restart'
)
ON CONFLICT (key) DO UPDATE 
SET value = EXCLUDED.value, 
    description = EXCLUDED.description;