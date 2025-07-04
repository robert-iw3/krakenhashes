-- Migration: Make agent_id nullable in job_tasks table
-- This allows job tasks to be created before being assigned to an agent,
-- which is necessary for rule splitting where chunks are pre-created

-- Make agent_id nullable
ALTER TABLE job_tasks 
ALTER COLUMN agent_id DROP NOT NULL;

-- Update the index to handle NULL values properly
DROP INDEX IF EXISTS idx_job_tasks_agent_status;
CREATE INDEX idx_job_tasks_agent_status ON job_tasks(agent_id, status) WHERE agent_id IS NOT NULL;

-- Add an index for finding unassigned tasks efficiently
CREATE INDEX idx_job_tasks_unassigned ON job_tasks(job_execution_id, status) WHERE agent_id IS NULL;