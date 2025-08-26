-- Add speedtest timeout setting to system_settings table
INSERT INTO system_settings (key, value, description, data_type)
VALUES (
    'speedtest_timeout_seconds', 
    '180', 
    'Maximum time to wait for speedtest completion (in seconds)', 
    'integer'
)
ON CONFLICT (key) DO NOTHING;