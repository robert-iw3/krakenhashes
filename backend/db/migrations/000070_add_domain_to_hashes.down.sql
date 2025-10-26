-- Rollback: Remove domain column from hashes table

DROP INDEX IF EXISTS idx_hashes_domain;
DROP INDEX IF EXISTS idx_hashes_username;

ALTER TABLE hashes
DROP COLUMN IF EXISTS domain;
