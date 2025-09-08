-- Remove reconnect grace period setting
DELETE FROM system_settings WHERE key = 'reconnect_grace_period_minutes';