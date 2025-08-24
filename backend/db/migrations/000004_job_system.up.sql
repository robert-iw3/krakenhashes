-- Create preset_jobs table (without is_small_job column which was removed later)
CREATE TABLE preset_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    wordlist_ids JSONB NOT NULL DEFAULT '[]',
    rule_ids JSONB NOT NULL DEFAULT '[]',
    attack_mode INTEGER NOT NULL DEFAULT 0 CHECK (attack_mode IN (0, 1, 3, 6, 7, 9)),
    priority INTEGER NOT NULL,
    chunk_size_seconds INTEGER NOT NULL,
    status_updates_enabled BOOLEAN NOT NULL DEFAULT true,
    allow_high_priority_override BOOLEAN NOT NULL DEFAULT false,
    binary_version_id INTEGER REFERENCES binary_versions(id) NOT NULL,
    mask TEXT DEFAULT NULL,
    keyspace BIGINT,
    max_agents INTEGER DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE preset_jobs IS 'Stores predefined job configurations for attacks';
COMMENT ON COLUMN preset_jobs.keyspace IS 'Pre-calculated keyspace for this preset job';
COMMENT ON COLUMN preset_jobs.max_agents IS 'Maximum number of agents that can work on jobs from this preset';

-- Create job_workflows table
CREATE TABLE job_workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create job_workflow_steps table
CREATE TABLE job_workflow_steps (
    id BIGSERIAL PRIMARY KEY,
    job_workflow_id UUID NOT NULL REFERENCES job_workflows(id) ON DELETE CASCADE,
    preset_job_id UUID NOT NULL REFERENCES preset_jobs(id),
    step_order INTEGER NOT NULL,
    UNIQUE (job_workflow_id, step_order)
);

-- Create job_executions table with all final columns
CREATE TABLE job_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preset_job_id UUID REFERENCES preset_jobs(id),
    hashlist_id BIGINT NOT NULL REFERENCES hashlists(id) ON DELETE CASCADE,
    job_workflow_id UUID REFERENCES job_workflows(id),
    workflow_step INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL,
    attack_mode INTEGER NOT NULL DEFAULT 0,
    mask TEXT,
    wordlist_ids JSONB DEFAULT '[]',
    rule_ids JSONB DEFAULT '[]',
    chunk_size_seconds INTEGER NOT NULL DEFAULT 1200,
    total_keyspace BIGINT,
    effective_keyspace BIGINT,
    dispatched_keyspace BIGINT DEFAULT 0,
    uses_rule_splitting BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    progress_percentage NUMERIC(5,2) DEFAULT 0.00,
    created_by UUID REFERENCES users(id),
    CONSTRAINT job_executions_status_check CHECK (status IN ('pending', 'queued', 'running', 'paused', 'completed', 'failed', 'cancelled', 'interrupted'))
);

COMMENT ON TABLE job_executions IS 'Tracks the execution of jobs against hashlists';
COMMENT ON COLUMN job_executions.effective_keyspace IS 'The actual keyspace being processed (may differ from total for rule-split jobs)';
COMMENT ON COLUMN job_executions.dispatched_keyspace IS 'Total keyspace that has been dispatched to agents';
COMMENT ON COLUMN job_executions.uses_rule_splitting IS 'Whether this job uses rule splitting for chunk distribution';

-- Create job_tasks table with all columns including later additions
CREATE TABLE job_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_execution_id UUID NOT NULL REFERENCES job_executions(id) ON DELETE CASCADE,
    agent_id INTEGER REFERENCES agents(id),
    chunk_number INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    detailed_status VARCHAR(50) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    crack_count INTEGER DEFAULT 0,
    skip BIGINT,
    "limit" BIGINT,
    rule_skip BIGINT,
    rule_limit BIGINT,
    effective_keyspace BIGINT,
    assigned_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    output TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    progress JSONB DEFAULT '{}'::jsonb,
    consecutive_failures INTEGER DEFAULT 0,
    CONSTRAINT valid_task_status CHECK (
        status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled', 'reconnect_pending') AND
        detailed_status IN ('pending', 'dispatched', 'running', 'completed_with_cracks', 'completed_no_cracks', 'failed', 'cancelled')
    )
);

COMMENT ON TABLE job_tasks IS 'Tracks individual chunks of work assigned to agents';
COMMENT ON COLUMN job_tasks.chunk_number IS 'Sequential chunk number for ordering and tracking';
COMMENT ON COLUMN job_tasks.effective_keyspace IS 'The actual keyspace size for this task';
COMMENT ON COLUMN job_tasks.consecutive_failures IS 'Number of consecutive failures for this task';

-- Create job_task_performance_metrics table
CREATE TABLE job_task_performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_task_id UUID NOT NULL REFERENCES job_tasks(id) ON DELETE CASCADE,
    agent_id INTEGER NOT NULL REFERENCES agents(id),
    device_id INTEGER,
    device_name VARCHAR(255),
    hashrate BIGINT,
    exec_runtime_ms BIGINT,
    estimated_time_remaining_seconds BIGINT,
    progress_percentage NUMERIC(5,2),
    hashes_processed BIGINT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE job_task_performance_metrics IS 'Stores performance metrics for job tasks with device tracking';

-- Create indexes
CREATE INDEX idx_job_workflow_steps_job_workflow_id ON job_workflow_steps(job_workflow_id);
CREATE INDEX idx_job_workflow_steps_preset_job_id ON job_workflow_steps(preset_job_id);
CREATE INDEX idx_job_executions_preset_job_id ON job_executions(preset_job_id);
CREATE INDEX idx_job_executions_hashlist_id ON job_executions(hashlist_id);
CREATE INDEX idx_job_executions_job_workflow_id ON job_executions(job_workflow_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_priority ON job_executions(priority);
CREATE INDEX idx_job_executions_created_at ON job_executions(created_at);
CREATE INDEX idx_job_executions_created_by ON job_executions(created_by);
CREATE INDEX idx_job_tasks_job_execution_id ON job_tasks(job_execution_id);
CREATE INDEX idx_job_tasks_agent_id ON job_tasks(agent_id);
CREATE INDEX idx_job_tasks_status ON job_tasks(status);
CREATE INDEX idx_job_tasks_detailed_status ON job_tasks(detailed_status);
CREATE INDEX idx_job_tasks_retry_count ON job_tasks(retry_count);
CREATE INDEX idx_job_tasks_chunk_number ON job_tasks(chunk_number);
CREATE INDEX idx_job_task_performance_metrics_job_task_id ON job_task_performance_metrics(job_task_id);
CREATE INDEX idx_job_task_performance_metrics_agent_id ON job_task_performance_metrics(agent_id);
CREATE INDEX idx_job_task_performance_metrics_timestamp ON job_task_performance_metrics(timestamp);

-- Create triggers
CREATE TRIGGER update_preset_jobs_updated_at
    BEFORE UPDATE ON preset_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_workflows_updated_at
    BEFORE UPDATE ON job_workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_tasks_updated_at
    BEFORE UPDATE ON job_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();