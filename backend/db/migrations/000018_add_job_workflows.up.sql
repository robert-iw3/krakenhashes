-- Enable UUID generation if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create the preset_jobs table
CREATE TABLE preset_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    wordlist_ids JSONB NOT NULL DEFAULT '[]',
    rule_ids JSONB NOT NULL DEFAULT '[]',
    attack_mode INTEGER NOT NULL DEFAULT 0 CHECK (attack_mode IN (0, 1, 3, 6, 7, 9)),
    priority INTEGER NOT NULL,
    chunk_size_seconds INTEGER NOT NULL,
    status_updates_enabled BOOLEAN NOT NULL DEFAULT true,
    is_small_job BOOLEAN NOT NULL DEFAULT false,
    allow_high_priority_override BOOLEAN NOT NULL DEFAULT false,
    binary_version_id INTEGER REFERENCES binary_versions(id) NOT NULL,
    mask TEXT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create the job_workflows table
CREATE TABLE job_workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create the job_workflow_steps table
CREATE TABLE job_workflow_steps (
    id BIGSERIAL PRIMARY KEY,
    job_workflow_id UUID NOT NULL REFERENCES job_workflows(id) ON DELETE CASCADE,
    preset_job_id UUID NOT NULL REFERENCES preset_jobs(id),
    step_order INTEGER NOT NULL,
    UNIQUE (job_workflow_id, step_order)
);

-- Add indexes for foreign keys in job_workflow_steps
CREATE INDEX idx_job_workflow_steps_job_workflow_id ON job_workflow_steps(job_workflow_id);
CREATE INDEX idx_job_workflow_steps_preset_job_id ON job_workflow_steps(preset_job_id);

-- Trigger to update updated_at timestamp on preset_jobs table
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_preset_jobs_updated_at
BEFORE UPDATE ON preset_jobs
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update updated_at timestamp on job_workflows table
CREATE TRIGGER update_job_workflows_updated_at
BEFORE UPDATE ON job_workflows
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column(); 