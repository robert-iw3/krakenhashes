-- Drop indexes first
DROP INDEX IF EXISTS idx_agent_hashlists_cleanup;
DROP INDEX IF EXISTS idx_job_metrics_lookup;
DROP INDEX IF EXISTS idx_agent_metrics_aggregation;
DROP INDEX IF EXISTS idx_agent_metrics_lookup;
DROP INDEX IF EXISTS idx_agent_benchmarks_lookup;
DROP INDEX IF EXISTS idx_job_tasks_execution;
DROP INDEX IF EXISTS idx_job_tasks_agent_status;
DROP INDEX IF EXISTS idx_job_executions_priority;
DROP INDEX IF EXISTS idx_job_executions_status;

-- Drop tables in reverse order to handle foreign key dependencies
DROP TABLE IF EXISTS agent_hashlists;
DROP TABLE IF EXISTS job_performance_metrics;
DROP TABLE IF EXISTS agent_performance_metrics;
DROP TABLE IF EXISTS agent_benchmarks;
DROP TABLE IF EXISTS job_tasks;
DROP TABLE IF EXISTS job_executions;