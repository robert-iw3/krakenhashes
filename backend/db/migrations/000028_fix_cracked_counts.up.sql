-- Fix cracked_hashes count in hashlists table by counting actual cracked hashes
UPDATE hashlists h
SET cracked_hashes = (
    SELECT COUNT(*)
    FROM hashes ha
    JOIN hashlist_hashes hh ON ha.id = hh.hash_id
    WHERE hh.hashlist_id = h.id AND ha.is_cracked = true
)
WHERE h.id IN (
    SELECT DISTINCT hh.hashlist_id
    FROM hashlist_hashes hh
    JOIN hashes ha ON ha.id = hh.hash_id
    WHERE ha.is_cracked = true
);

-- Update crack_count in job_tasks table for completed tasks
UPDATE job_tasks jt
SET crack_count = (
    SELECT COUNT(*)
    FROM hashes h
    JOIN hashlist_hashes hh ON h.id = hh.hash_id
    JOIN job_executions je ON je.hashlist_id = hh.hashlist_id
    WHERE je.id = jt.job_execution_id
    AND h.is_cracked = true
    AND h.last_updated >= jt.assigned_at
    AND (jt.completed_at IS NULL OR h.last_updated <= jt.completed_at)
)
WHERE jt.status = 'completed'
AND EXISTS (
    SELECT 1
    FROM job_executions je
    JOIN hashlist_hashes hh ON hh.hashlist_id = je.hashlist_id
    JOIN hashes h ON h.id = hh.hash_id
    WHERE je.id = jt.job_execution_id
    AND h.is_cracked = true
);