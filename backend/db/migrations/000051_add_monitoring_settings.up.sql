-- Add monitoring-related settings to system_settings table
INSERT INTO system_settings (key, value, description, data_type, updated_at)
VALUES 
    ('enable_aggregation', 'true', 'Enable metrics aggregation for long-term storage', 'boolean', NOW()),
    ('aggregation_interval', 'daily', 'Interval for metrics aggregation (hourly, daily, weekly)', 'string', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    data_type = EXCLUDED.data_type,
    updated_at = NOW();