-- Remove task heartbeat timeout setting
DELETE FROM system_settings WHERE key = 'task_heartbeat_timeout_minutes';