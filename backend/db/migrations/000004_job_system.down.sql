-- Drop triggers
DROP TRIGGER IF EXISTS update_job_tasks_updated_at ON job_tasks;
DROP TRIGGER IF EXISTS update_job_workflows_updated_at ON job_workflows;
DROP TRIGGER IF EXISTS update_preset_jobs_updated_at ON preset_jobs;

-- Drop indexes
DROP INDEX IF EXISTS idx_job_task_performance_metrics_timestamp;
DROP INDEX IF EXISTS idx_job_task_performance_metrics_agent_id;
DROP INDEX IF EXISTS idx_job_task_performance_metrics_job_task_id;
DROP INDEX IF EXISTS idx_job_tasks_chunk_number;
DROP INDEX IF EXISTS idx_job_tasks_retry_count;
DROP INDEX IF EXISTS idx_job_tasks_detailed_status;
DROP INDEX IF EXISTS idx_job_tasks_status;
DROP INDEX IF EXISTS idx_job_tasks_agent_id;
DROP INDEX IF EXISTS idx_job_tasks_job_execution_id;
DROP INDEX IF EXISTS idx_job_executions_created_by;
DROP INDEX IF EXISTS idx_job_executions_created_at;
DROP INDEX IF EXISTS idx_job_executions_priority;
DROP INDEX IF EXISTS idx_job_executions_status;
DROP INDEX IF EXISTS idx_job_executions_job_workflow_id;
DROP INDEX IF EXISTS idx_job_executions_hashlist_id;
DROP INDEX IF EXISTS idx_job_executions_preset_job_id;
DROP INDEX IF EXISTS idx_job_workflow_steps_preset_job_id;
DROP INDEX IF EXISTS idx_job_workflow_steps_job_workflow_id;

-- Drop tables
DROP TABLE IF EXISTS job_task_performance_metrics;
DROP TABLE IF EXISTS job_tasks;
DROP TABLE IF EXISTS job_executions;
DROP TABLE IF EXISTS job_workflow_steps;
DROP TABLE IF EXISTS job_workflows;
DROP TABLE IF EXISTS preset_jobs;