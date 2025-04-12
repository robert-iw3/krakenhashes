package queries

// --- Client Query Constants ---

const CreateClientQuery = `
INSERT INTO clients (id, name, description, contact_info, data_retention_months, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

const GetClientByIDQuery = `
SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
FROM clients
WHERE id = $1
`

const ListClientsQuery = `
SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
FROM clients
ORDER BY name ASC
`

const UpdateClientQuery = `
UPDATE clients
SET name = $1, description = $2, contact_info = $3, data_retention_months = $4, updated_at = $5
WHERE id = $6
`

const DeleteClientQuery = `DELETE FROM clients WHERE id = $1`

const GetClientByNameQuery = `
SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
FROM clients
WHERE name = $1
`

const SearchClientsQuery = `
SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
FROM clients
WHERE name ILIKE $1 OR description ILIKE $1
ORDER BY name ASC
LIMIT 50
`
