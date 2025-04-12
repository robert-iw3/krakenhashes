package queries

// --- ClientSettings Query Constants ---

const GetClientSettingQuery = `
SELECT key, value, description, updated_at
FROM client_settings
WHERE key = $1
`

const SetClientSettingQuery = `
UPDATE client_settings
SET value = $1, updated_at = $2
WHERE key = $3
`

const GetAllClientSettingsQuery = `
SELECT key, value, description, updated_at
FROM client_settings
ORDER BY key ASC
`
