-- Enhance job_tasks table with detailed chunk tracking
ALTER TABLE job_tasks 
ADD COLUMN crack_count INTEGER DEFAULT 0,
ADD COLUMN detailed_status VARCHAR(50) DEFAULT 'pending',
ADD COLUMN retry_count INTEGER DEFAULT 0;

-- Update existing status constraint to allow new detailed status values
ALTER TABLE job_tasks 
DROP CONSTRAINT valid_task_status;

ALTER TABLE job_tasks 
ADD CONSTRAINT valid_task_status CHECK (
    status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled') AND
    detailed_status IN ('pending', 'dispatched', 'running', 'completed_with_cracks', 'completed_no_cracks', 'failed', 'cancelled')
);

-- Add new system settings for job management
INSERT INTO system_settings (key, value, description, data_type)
VALUES 
    ('job_refresh_interval_seconds', '5', 'Interval in seconds for refreshing job status in the UI', 'integer'),
    ('max_chunk_retry_attempts', '3', 'Maximum number of retry attempts for failed job chunks', 'integer'),
    ('jobs_per_page_default', '25', 'Default number of jobs to display per page in the UI', 'integer')
ON CONFLICT (key) DO NOTHING;

-- Create index for efficient chunk status queries
CREATE INDEX IF NOT EXISTS idx_job_tasks_detailed_status ON job_tasks(detailed_status);
CREATE INDEX IF NOT EXISTS idx_job_tasks_retry_count ON job_tasks(retry_count);