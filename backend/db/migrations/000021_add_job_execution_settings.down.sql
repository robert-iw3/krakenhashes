-- Remove job execution related system settings
DELETE FROM system_settings 
WHERE key IN (
    'default_chunk_duration',
    'chunk_fluctuation_percentage',
    'agent_hashlist_retention_hours',
    'progress_reporting_interval',
    'max_concurrent_jobs_per_agent',
    'job_interruption_enabled',
    'benchmark_cache_duration_hours',
    'enable_realtime_crack_notifications',
    'metrics_retention_realtime_days',
    'metrics_retention_daily_days',
    'metrics_retention_weekly_days'
);