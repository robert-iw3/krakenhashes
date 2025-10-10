-- Remove potfile exclusion flag from hashlists table

DROP INDEX IF EXISTS idx_hashlists_exclude_from_potfile;
ALTER TABLE hashlists DROP COLUMN IF EXISTS exclude_from_potfile;
