-- Add job execution related system settings
INSERT INTO system_settings (key, value, description, data_type)
VALUES 
    ('default_chunk_duration', '1200', 'Default duration in seconds for job chunks (default: 20 minutes)', 'integer'),
    ('chunk_fluctuation_percentage', '20', 'Percentage fluctuation allowed for final job chunks to avoid small remainder chunks', 'integer'),
    ('agent_hashlist_retention_hours', '24', 'Hours to retain hashlists on agents before cleanup (default: 24 hours)', 'integer'),
    ('progress_reporting_interval', '5', 'Interval in seconds for agents to report job progress (default: 5 seconds)', 'integer'),
    ('max_concurrent_jobs_per_agent', '1', 'Maximum number of concurrent jobs an agent can process', 'integer'),
    ('job_interruption_enabled', 'true', 'Whether higher priority jobs can interrupt running jobs', 'boolean'),
    ('benchmark_cache_duration_hours', '168', 'Hours to cache agent benchmark results before re-running (default: 7 days)', 'integer'),
    ('enable_realtime_crack_notifications', 'true', 'Enable real-time notifications when hashes are cracked', 'boolean'),
    ('metrics_retention_realtime_days', '7', 'Days to retain real-time metrics before aggregation', 'integer'),
    ('metrics_retention_daily_days', '30', 'Days to retain daily aggregated metrics before weekly aggregation', 'integer'),
    ('metrics_retention_weekly_days', '365', 'Days to retain weekly aggregated metrics', 'integer');