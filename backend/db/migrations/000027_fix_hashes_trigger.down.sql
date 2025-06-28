-- Restore the old (incorrect) trigger
DROP TRIGGER IF EXISTS update_hashes_last_updated ON hashes;

CREATE TRIGGER update_hashes_last_updated
BEFORE UPDATE ON hashes
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Drop the new function
DROP FUNCTION IF EXISTS update_last_updated_column();