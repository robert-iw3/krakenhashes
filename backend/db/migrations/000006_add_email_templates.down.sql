-- Remove initial email templates
DELETE FROM email_templates 
WHERE template_type IN ('security_event', 'job_completion', 'admin_error', 'mfa_code'); 