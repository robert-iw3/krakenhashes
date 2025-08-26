-- Add task heartbeat timeout setting (default 5 minutes)
INSERT INTO system_settings (key, value, description) 
VALUES (
    'task_heartbeat_timeout_minutes', 
    '5', 
    'Timeout in minutes for task heartbeat. Tasks without heartbeat for this duration will be reset to pending.'
)
ON CONFLICT (key) DO UPDATE 
SET value = EXCLUDED.value,
    description = EXCLUDED.description;