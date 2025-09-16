-- Add agent file transfer settings to the system_settings table
-- These settings control how agents download files from the backend

-- Insert default agent download settings
INSERT INTO system_settings (key, value, description, data_type, updated_at)
VALUES
    ('agent_max_concurrent_downloads', '3', 'Maximum number of concurrent file downloads per agent', 'integer', NOW()),
    ('agent_download_timeout_minutes', '60', 'Timeout in minutes for file downloads', 'integer', NOW()),
    ('agent_download_retry_attempts', '3', 'Number of retry attempts for failed downloads', 'integer', NOW()),
    ('agent_download_progress_interval_seconds', '10', 'Interval in seconds for progress reporting', 'integer', NOW()),
    ('agent_download_chunk_size_mb', '10', 'Download chunk size in megabytes for resume capability', 'integer', NOW())
ON CONFLICT (key) DO NOTHING;