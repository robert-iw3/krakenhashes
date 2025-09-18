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

// ListClientsWithCrackedCountsQuery retrieves all clients with their cracked hash counts
const ListClientsWithCrackedCountsQuery = `
SELECT
    c.id,
    c.name,
    c.description,
    c.contact_info,
    c.data_retention_months,
    c.created_at,
    c.updated_at,
    COUNT(DISTINCT h.id) FILTER (WHERE h.is_cracked = true) as cracked_count
FROM clients c
LEFT JOIN hashlists hl ON hl.client_id = c.id
LEFT JOIN hashlist_hashes hh ON hh.hashlist_id = hl.id
LEFT JOIN hashes h ON h.id = hh.hash_id
GROUP BY c.id, c.name, c.description, c.contact_info, c.data_retention_months, c.created_at, c.updated_at
ORDER BY c.name ASC
`
