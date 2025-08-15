-- Revert reconnect_pending status addition

-- First, update any tasks with reconnect_pending status back to pending
UPDATE job_tasks SET status = 'pending' WHERE status = 'reconnect_pending';

-- Drop the constraint
ALTER TABLE job_tasks DROP CONSTRAINT IF EXISTS valid_task_status;

-- Re-add the original constraint without reconnect_pending
ALTER TABLE job_tasks ADD CONSTRAINT valid_task_status CHECK (
    status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled')
    AND detailed_status IN ('pending', 'dispatched', 'running', 'completed_with_cracks', 'completed_no_cracks', 'failed', 'cancelled')
);

-- Restore original comment
COMMENT ON COLUMN job_tasks.status IS 'Task status: pending, assigned, running, completed, failed, cancelled';