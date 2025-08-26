-- Add created_at column to job_tasks table if it doesn't exist
ALTER TABLE job_tasks ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Update existing rows to have created_at if null
UPDATE job_tasks SET created_at = COALESCE(assigned_at, CURRENT_TIMESTAMP) WHERE created_at IS NULL;

-- Make created_at NOT NULL after setting values
ALTER TABLE job_tasks ALTER COLUMN created_at SET NOT NULL;