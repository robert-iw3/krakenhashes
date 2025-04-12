package queries

// --- Hash Query Constants (Transactional) ---

const GetHashIDsByHashlistIDQuery = `SELECT hash_id FROM hashlist_hashes WHERE hashlist_id = $1`

const DeleteHashlistAssociationsQuery = `DELETE FROM hashlist_hashes WHERE hashlist_id = $1`

const CheckHashAssociationExistsQuery = `SELECT EXISTS (SELECT 1 FROM hashlist_hashes WHERE hash_id = $1)`

const DeleteHashByIDQuery = `DELETE FROM hashes WHERE id = $1`
