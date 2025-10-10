-- Remove potfile exclusion flag from clients table

DROP INDEX IF EXISTS idx_clients_exclude_from_potfile;
ALTER TABLE clients DROP COLUMN IF EXISTS exclude_from_potfile;
