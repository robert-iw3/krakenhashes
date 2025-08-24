-- Delete hash types
DELETE FROM hash_types WHERE id IN (0, 100, 1000, 1400, 1700, 22000);

-- Delete email templates
DELETE FROM email_templates WHERE template_type IN ('security_event', 'job_completion', 'admin_error', 'mfa_code');

-- Delete system settings
DELETE FROM system_settings WHERE key IN (
    'max_job_priority',
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
    'metrics_retention_weekly_days',
    'job_refresh_interval_seconds',
    'max_chunk_retry_attempts',
    'jobs_per_page_default',
    'job_interruption_priority_threshold',
    'job_interruption_grace_period_seconds',
    'enable_dynamic_chunking',
    'min_chunk_duration_seconds',
    'max_chunk_duration_seconds',
    'enable_keyspace_splitting',
    'enable_rule_splitting',
    'max_chunks_per_job',
    'speedtest_timeout_seconds',
    'scheduler_enabled',
    'scheduler_check_interval_seconds',
    'enable_agent_scheduling',
    'default_agent_priority',
    'task_heartbeat_timeout_seconds',
    'potfile_auto_import',
    'potfile_sync_interval_minutes',
    'potfile_retention_days',
    'enable_performance_monitoring',
    'metrics_collection_interval_seconds',
    'alert_on_agent_disconnect',
    'agent_heartbeat_timeout_seconds',
    'enable_resource_usage_alerts',
    'cpu_usage_alert_threshold',
    'gpu_temp_alert_threshold',
    'memory_usage_alert_threshold'
);

-- Delete client settings
DELETE FROM client_settings WHERE key = 'default_data_retention_months';

-- Delete auth settings
DELETE FROM auth_settings WHERE setting_key IN (
    'password_min_length',
    'password_require_uppercase',
    'password_require_lowercase',
    'password_require_number',
    'password_require_special',
    'password_history_count',
    'password_expiry_days',
    'session_timeout_minutes',
    'session_absolute_timeout_hours',
    'max_login_attempts',
    'lockout_duration_minutes',
    'mfa_required',
    'mfa_grace_period_days',
    'api_rate_limit_per_minute',
    'allow_registration',
    'require_email_verification',
    'password_reset_token_expiry_hours',
    'remember_me_duration_days'
);

-- Delete default users (keep system user as it may be referenced)
DELETE FROM users WHERE id = 'f1a50a5d-ff43-4617-bc21-61c54b214dee';
-- Note: System user (00000000-0000-0000-0000-000000000000) is kept as it may have references