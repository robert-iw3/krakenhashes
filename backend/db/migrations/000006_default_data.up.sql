-- Insert essential users only (no test users)
DO $$ 
DECLARE
    admin_id UUID := 'f1a50a5d-ff43-4617-bc21-61c54b214dee'; -- Hardcoded admin UUID
    system_id UUID := '00000000-0000-0000-0000-000000000000'; -- System user with UUID.Nil
BEGIN
    -- Insert system user first (cannot be used for login)
    INSERT INTO users (id, username, email, password_hash, role, status)
    VALUES (system_id, 'system', 'system@krakenhashes.local', 'SYSTEM_USER_NO_LOGIN', 'system', 'active')
    ON CONFLICT (id) DO NOTHING;

    -- Insert admin user for initial login
    INSERT INTO users (id, username, first_name, last_name, email, password_hash, role)
    VALUES (admin_id, 'admin', 'Admin', 'User', 'admin@example.com', '$2a$10$2gobOj6ATVGUNNk5CHw9de2reYqSZVHtP/Qrx63.Ho9nTWbo5PW7O', 'admin') -- password: KrakenHashes1!
    ON CONFLICT (id) DO NOTHING;
END $$;

-- Insert auth settings
INSERT INTO auth_settings (
    setting_key, 
    setting_value, 
    setting_type, 
    description,
    is_encrypted,
    created_at,
    updated_at
)
VALUES 
    ('password_min_length', '12', 'integer', 'Minimum password length required', false, NOW(), NOW()),
    ('password_require_uppercase', 'true', 'boolean', 'Require at least one uppercase letter', false, NOW(), NOW()),
    ('password_require_lowercase', 'true', 'boolean', 'Require at least one lowercase letter', false, NOW(), NOW()),
    ('password_require_number', 'true', 'boolean', 'Require at least one number', false, NOW(), NOW()),
    ('password_require_special', 'true', 'boolean', 'Require at least one special character', false, NOW(), NOW()),
    ('password_history_count', '5', 'integer', 'Number of previous passwords to check against', false, NOW(), NOW()),
    ('password_expiry_days', '90', 'integer', 'Days until password expires (0 = never)', false, NOW(), NOW()),
    ('session_timeout_minutes', '30', 'integer', 'Minutes of inactivity before session expires', false, NOW(), NOW()),
    ('session_absolute_timeout_hours', '24', 'integer', 'Maximum session duration in hours', false, NOW(), NOW()),
    ('max_login_attempts', '5', 'integer', 'Maximum failed login attempts before lockout', false, NOW(), NOW()),
    ('lockout_duration_minutes', '15', 'integer', 'Account lockout duration in minutes', false, NOW(), NOW()),
    ('mfa_required', 'false', 'boolean', 'Require MFA for all users', false, NOW(), NOW()),
    ('mfa_grace_period_days', '7', 'integer', 'Days to allow login without MFA after enabling', false, NOW(), NOW()),
    ('api_rate_limit_per_minute', '60', 'integer', 'API requests allowed per minute per user', false, NOW(), NOW()),
    ('allow_registration', 'false', 'boolean', 'Allow new user registration', false, NOW(), NOW()),
    ('require_email_verification', 'true', 'boolean', 'Require email verification for new accounts', false, NOW(), NOW()),
    ('password_reset_token_expiry_hours', '24', 'integer', 'Hours until password reset token expires', false, NOW(), NOW()),
    ('remember_me_duration_days', '30', 'integer', 'Days to keep "Remember Me" session active', false, NOW(), NOW())
ON CONFLICT (setting_key) DO NOTHING;

-- Insert client settings
INSERT INTO client_settings (key, value, description)
VALUES ('default_data_retention_months', '0', 'Default data retention period in months for clients without a specific setting. 0 means keep forever.')
ON CONFLICT (key) DO NOTHING;

-- Insert system settings
INSERT INTO system_settings (key, value, description, data_type)
VALUES 
    -- Core settings
    ('max_job_priority', '1000', 'Maximum priority value that can be assigned to jobs and preset jobs', 'integer'),
    
    -- Job execution settings
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
    ('metrics_retention_weekly_days', '365', 'Days to retain weekly aggregated metrics', 'integer'),
    
    -- Job management settings
    ('job_refresh_interval_seconds', '5', 'Interval in seconds for refreshing job status in the UI', 'integer'),
    ('max_chunk_retry_attempts', '3', 'Maximum number of retry attempts for failed job chunks', 'integer'),
    ('jobs_per_page_default', '25', 'Default number of jobs to display per page in the UI', 'integer'),
    
    -- Interruption settings
    ('job_interruption_priority_threshold', '500', 'Minimum priority difference required to interrupt a running job', 'integer'),
    ('job_interruption_grace_period_seconds', '30', 'Grace period in seconds before interrupting a job', 'integer'),
    
    -- Chunking settings
    ('enable_dynamic_chunking', 'true', 'Enable dynamic chunk size adjustment based on agent performance', 'boolean'),
    ('min_chunk_duration_seconds', '300', 'Minimum chunk duration in seconds', 'integer'),
    ('max_chunk_duration_seconds', '7200', 'Maximum chunk duration in seconds', 'integer'),
    ('enable_keyspace_splitting', 'true', 'Enable splitting jobs by keyspace for better distribution', 'boolean'),
    ('enable_rule_splitting', 'true', 'Enable splitting jobs by rules for better distribution', 'boolean'),
    ('max_chunks_per_job', '1000', 'Maximum number of chunks a single job can be split into', 'integer'),
    
    -- Speedtest settings
    ('speedtest_timeout_seconds', '30', 'Timeout in seconds for agent speedtest operations', 'integer'),
    
    -- Scheduling settings
    ('scheduler_enabled', 'true', 'Enable the job scheduler', 'boolean'),
    ('scheduler_check_interval_seconds', '10', 'How often the scheduler checks for new jobs', 'integer'),
    ('enable_agent_scheduling', 'true', 'Enable agent-specific scheduling preferences', 'boolean'),
    ('default_agent_priority', 'medium', 'Default priority level for agents without specific settings', 'string'),
    
    -- Task heartbeat settings
    ('task_heartbeat_timeout_seconds', '60', 'Seconds without heartbeat before considering a task stalled', 'integer'),
    
    -- Potfile settings
    ('potfile_auto_import', 'true', 'Automatically import potfile entries when agents crack hashes', 'boolean'),
    ('potfile_sync_interval_minutes', '60', 'Interval in minutes to sync potfile entries between agents', 'integer'),
    ('potfile_retention_days', '0', 'Days to retain potfile entries (0 = forever)', 'integer'),
    
    -- Monitoring settings
    ('enable_performance_monitoring', 'true', 'Enable detailed performance monitoring and metrics collection', 'boolean'),
    ('metrics_collection_interval_seconds', '10', 'Interval for collecting performance metrics from agents', 'integer'),
    ('alert_on_agent_disconnect', 'true', 'Send alerts when agents disconnect unexpectedly', 'boolean'),
    ('agent_heartbeat_timeout_seconds', '30', 'Seconds without heartbeat before marking agent as disconnected', 'integer'),
    ('enable_resource_usage_alerts', 'true', 'Enable alerts for high resource usage on agents', 'boolean'),
    ('cpu_usage_alert_threshold', '90', 'CPU usage percentage threshold for alerts', 'integer'),
    ('gpu_temp_alert_threshold', '85', 'GPU temperature threshold in Celsius for alerts', 'integer'),
    ('memory_usage_alert_threshold', '90', 'Memory usage percentage threshold for alerts', 'integer')
ON CONFLICT (key) DO NOTHING;

-- Add email templates
INSERT INTO email_templates (template_type, name, subject, html_content, text_content, created_at, updated_at)
VALUES
    -- Security Event Template
    ('security_event', 'Security Event Notification', 'Security Alert: {{ .EventType }}',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        .header {
            background-color: #000000;
            padding: 20px;
            text-align: center;
            width: 100%;
        }
        .header h1 {
            color: #FF0000;
            font-family: Arial, sans-serif;
            margin: 0;
        }
        .content {
            padding: 20px;
            font-family: Arial, sans-serif;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>KrakenHashes</h1>
    </div>
    <div class="content">
        <h2>Security Event Detected</h2>
        <p>Event Type: {{ .EventType }}</p>
        <p>Time: {{ .Timestamp }}</p>
        <p>Details: {{ .Details }}</p>
        <p>IP Address: {{ .IPAddress }}</p>
        <hr>
        <p>If you did not initiate this action, please contact support immediately.</p>
    </div>
</body>
</html>',
    'SECURITY EVENT ALERT

Event Type: {{ .EventType }}
Time: {{ .Timestamp }}
Details: {{ .Details }}
IP Address: {{ .IPAddress }}

If you did not initiate this action, please contact support immediately.',
    NOW(), NOW()),

    -- Job Completion Template
    ('job_completion', 'Job Completion Notification', 'Job Complete: {{ .JobName }}',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        .header {
            background-color: #000000;
            padding: 20px;
            text-align: center;
            width: 100%;
        }
        .header h1 {
            color: #FF0000;
            font-family: Arial, sans-serif;
            margin: 0;
        }
        .content {
            padding: 20px;
            font-family: Arial, sans-serif;
        }
        .stats {
            background-color: #f5f5f5;
            padding: 15px;
            border-radius: 5px;
            margin: 15px 0;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>KrakenHashes</h1>
    </div>
    <div class="content">
        <h2>Job Completed Successfully</h2>
        <p>Your job "{{ .JobName }}" has completed processing.</p>
        <div class="stats">
            <h3>Job Statistics</h3>
            <p>Duration: {{ .Duration }}</p>
            <p>Hashes Processed: {{ .HashesProcessed }}</p>
            <p>Cracked: {{ .CrackedCount }}</p>
            <p>Success Rate: {{ .SuccessRate }}%</p>
        </div>
        <p>View detailed results in your dashboard.</p>
    </div>
</body>
</html>',
    'JOB COMPLETION NOTIFICATION

Your job "{{ .JobName }}" has completed processing.

Job Statistics:
- Duration: {{ .Duration }}
- Hashes Processed: {{ .HashesProcessed }}
- Cracked: {{ .CrackedCount }}
- Success Rate: {{ .SuccessRate }}%

View detailed results in your dashboard.',
    NOW(), NOW()),

    -- Admin Error Template
    ('admin_error', 'System Error Alert', 'System Error: {{ .ErrorType }}',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        .header {
            background-color: #000000;
            padding: 20px;
            text-align: center;
            width: 100%;
        }
        .header h1 {
            color: #FF0000;
            font-family: Arial, sans-serif;
            margin: 0;
        }
        .content {
            padding: 20px;
            font-family: Arial, sans-serif;
        }
        .error-details {
            background-color: #fff1f0;
            border-left: 4px solid #ff4d4f;
            padding: 15px;
            margin: 15px 0;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>KrakenHashes</h1>
    </div>
    <div class="content">
        <h2>System Error Detected</h2>
        <div class="error-details">
            <p><strong>Error Type:</strong> {{ .ErrorType }}</p>
            <p><strong>Component:</strong> {{ .Component }}</p>
            <p><strong>Time:</strong> {{ .Timestamp }}</p>
            <p><strong>Error Message:</strong> {{ .ErrorMessage }}</p>
            <p><strong>Stack Trace:</strong> {{ .StackTrace }}</p>
        </div>
        <p>Please investigate and resolve this issue as soon as possible.</p>
    </div>
</body>
</html>',
    'SYSTEM ERROR ALERT

Error Type: {{ .ErrorType }}
Component: {{ .Component }}
Time: {{ .Timestamp }}
Error Message: {{ .ErrorMessage }}
Stack Trace: {{ .StackTrace }}

Please investigate and resolve this issue as soon as possible.',
    NOW(), NOW()),

    -- MFA Code Template
    ('mfa_code', 'Your Authentication Code', 'KrakenHashes Authentication Code',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        .header {
            background-color: #000000;
            padding: 20px;
            text-align: center;
            width: 100%;
        }
        .header h1 {
            color: #FF0000;
            font-family: Arial, sans-serif;
            margin: 0;
        }
        .content {
            padding: 20px;
            font-family: Arial, sans-serif;
            text-align: center;
        }
        .code {
            font-size: 32px;
            font-weight: bold;
            letter-spacing: 5px;
            color: #1a1a1a;
            background-color: #f5f5f5;
            padding: 20px;
            margin: 20px 0;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>KrakenHashes</h1>
    </div>
    <div class="content">
        <h2>Your Authentication Code</h2>
        <div class="code">{{ .Code }}</div>
        <p>This code will expire in {{ .ExpiryMinutes }} minutes.</p>
        <p>If you did not request this code, please ignore this email.</p>
    </div>
</body>
</html>',
    'Your KrakenHashes Authentication Code

Your code is: {{ .Code }}

This code will expire in {{ .ExpiryMinutes }} minutes.

If you did not request this code, please ignore this email.',
    NOW(), NOW())
ON CONFLICT (template_type) DO NOTHING;

-- Note: Hash types data should be loaded from a separate script or file
-- due to the large volume of data (100+ hash type entries).
-- See the original migration 000016_add_hashcat_hash_types.up.sql for the full list.
-- For initial setup, at minimum, add common hash types:

INSERT INTO hash_types (id, name, description, example, needs_processing, is_enabled, slow)
VALUES
    (0, 'MD5', NULL, '8743b52063cd84097a65d1633f5c74f5', FALSE, TRUE, FALSE),
    (100, 'SHA1', NULL, 'b89eaac7e61417341b710b727768294d0e6a277b', FALSE, TRUE, FALSE),
    (1000, 'NTLM', NULL, 'b4b9b02e6f09a9bd760f388b67351e2b', TRUE, TRUE, FALSE),
    (1400, 'SHA2-256', NULL, '127e6fbfe24a750e72930c220a8e138275656b8e5d8f48a98c3c92df2caba935', FALSE, TRUE, FALSE),
    (1700, 'SHA2-512', NULL, '82a9dda829eb7f8ffe9fbe49e45d47d2dad9664fbb7adf72492e3c81ebd3e29134d9bc12212bf83c6840638c40e9002e5c5a40c3a4a2836684725068f2a23f1c', FALSE, TRUE, FALSE),
    (22000, 'WPA-PBKDF2-PMKID+EAPOL', NULL, 'WPA*01*4d4fe7aac3a2cecab195321ceb99a7d0*fc690c158264*f4747f87f9f4*686173686361742d6573736964***', FALSE, TRUE, TRUE)
ON CONFLICT (id) DO NOTHING;