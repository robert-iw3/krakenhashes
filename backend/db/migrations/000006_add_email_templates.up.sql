-- Add initial email templates
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
    NOW(), NOW()); 