-- Remove agent download settings
DELETE FROM system_settings WHERE key IN (
    'agent_max_concurrent_downloads',
    'agent_download_timeout_minutes',
    'agent_download_retry_attempts',
    'agent_download_progress_interval_seconds',
    'agent_download_chunk_size_mb'
);