-- Remove require_client_for_hashlist setting
DELETE FROM system_settings WHERE key = 'require_client_for_hashlist';
