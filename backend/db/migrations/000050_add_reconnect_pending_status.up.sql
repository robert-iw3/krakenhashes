-- Add reconnect_pending status to job_tasks table
-- This status is used when an agent disconnects to give it a grace period to reconnect

-- Drop the existing constraint
ALTER TABLE job_tasks DROP CONSTRAINT IF EXISTS valid_task_status;

-- Add the new constraint with reconnect_pending included
ALTER TABLE job_tasks ADD CONSTRAINT valid_task_status CHECK (
    status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled', 'reconnect_pending')
    AND detailed_status IN ('pending', 'dispatched', 'running', 'completed_with_cracks', 'completed_no_cracks', 'failed', 'cancelled')
);

-- Add comment explaining the new status
COMMENT ON COLUMN job_tasks.status IS 'Task status: pending (waiting for assignment), assigned (assigned to agent), running (being executed), completed (finished successfully), failed (finished with error), cancelled (cancelled by user), reconnect_pending (agent disconnected, waiting for reconnection)';