-- Drop the incorrect trigger
DROP TRIGGER IF EXISTS update_hashes_last_updated ON hashes;

-- Create a proper function for updating last_updated column
CREATE OR REPLACE FUNCTION update_last_updated_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.last_updated = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the correct trigger to hashes table
CREATE TRIGGER update_hashes_last_updated
BEFORE UPDATE ON hashes
FOR EACH ROW
EXECUTE FUNCTION update_last_updated_column();