-- Add chunk_actual_keyspace to track actual keyspace size from hashcat progress[1]
-- This enables self-correcting cascade updates for subsequent chunks

ALTER TABLE job_tasks ADD COLUMN chunk_actual_keyspace BIGINT DEFAULT NULL;

COMMENT ON COLUMN job_tasks.chunk_actual_keyspace IS 'Actual keyspace size for this chunk from hashcat progress[1]. NULL until first progress update received. Used to recalculate subsequent chunks'' positions.';
