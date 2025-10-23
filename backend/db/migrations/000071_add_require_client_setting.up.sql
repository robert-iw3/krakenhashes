-- Add system setting to require client assignment for hashlists
INSERT INTO system_settings (key, value, description, data_type)
VALUES ('require_client_for_hashlist', 'false', 'Require client assignment when uploading new hashlists', 'boolean');
