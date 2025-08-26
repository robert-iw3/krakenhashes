-- Set owner_id to created_by_id for all existing agents where owner_id is NULL
UPDATE agents 
SET owner_id = created_by_id 
WHERE owner_id IS NULL;