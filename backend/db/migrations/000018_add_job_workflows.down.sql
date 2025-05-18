-- Drop the job_workflow_steps table first due to foreign key constraints
DROP TABLE IF EXISTS job_workflow_steps;

-- Drop the job_workflows table
DROP TABLE IF EXISTS job_workflows;

-- Drop the preset_jobs table
DROP TABLE IF EXISTS preset_jobs;

-- Note: The update_updated_at_column function and the uuid-ossp extension are not dropped
-- as they might be used by other parts of the schema. 