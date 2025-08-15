-- Remove monitoring-related settings from system_settings table
DELETE FROM system_settings WHERE key IN ('enable_aggregation', 'aggregation_interval');