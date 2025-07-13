-- Remove speedtest timeout setting from system_settings table
DELETE FROM system_settings WHERE key = 'speedtest_timeout_seconds';