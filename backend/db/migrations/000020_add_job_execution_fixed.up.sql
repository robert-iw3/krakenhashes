-- Create job_executions table to track actual job runs
CREATE TABLE IF NOT EXISTS job_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preset_job_id UUID NOT NULL REFERENCES preset_jobs(id) ON DELETE CASCADE,
    hashlist_id BIGINT NOT NULL REFERENCES hashlists(id) ON DELETE CASCADE,  -- Changed from UUID to BIGINT
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 0,
    total_keyspace BIGINT,
    processed_keyspace BIGINT DEFAULT 0,
    attack_mode INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    interrupted_by UUID REFERENCES job_executions(id),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled'))
);

-- Create job_tasks table for individual chunks assigned to agents
CREATE TABLE IF NOT EXISTS job_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_execution_id UUID NOT NULL REFERENCES job_executions(id) ON DELETE CASCADE,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,  -- Changed from UUID to INTEGER
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    keyspace_start BIGINT NOT NULL,
    keyspace_end BIGINT NOT NULL,
    keyspace_processed BIGINT DEFAULT 0,
    benchmark_speed BIGINT, -- hashes per second
    chunk_duration INT NOT NULL, -- seconds
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_checkpoint TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    CONSTRAINT valid_task_status CHECK (status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'cancelled'))
);

-- Create agent_benchmarks table to store benchmark results
CREATE TABLE IF NOT EXISTS agent_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,  -- Changed from UUID to INTEGER
    attack_mode INT NOT NULL,
    hash_type INT NOT NULL,
    speed BIGINT NOT NULL, -- hashes per second
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, attack_mode, hash_type)
);

-- Create agent_performance_metrics table for historical tracking
CREATE TABLE IF NOT EXISTS agent_performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,  -- Changed from UUID to INTEGER
    metric_type VARCHAR(50) NOT NULL,
    value NUMERIC NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    aggregation_level VARCHAR(20) NOT NULL DEFAULT 'realtime',
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    CONSTRAINT valid_metric_type CHECK (metric_type IN ('hash_rate', 'utilization', 'temperature', 'power_usage')),
    CONSTRAINT valid_aggregation CHECK (aggregation_level IN ('realtime', 'daily', 'weekly'))
);

-- Create job_performance_metrics table for job tracking
CREATE TABLE IF NOT EXISTS job_performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_execution_id UUID NOT NULL REFERENCES job_executions(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    value NUMERIC NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    aggregation_level VARCHAR(20) NOT NULL DEFAULT 'realtime',
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    CONSTRAINT valid_job_metric_type CHECK (metric_type IN ('hash_rate', 'progress_percentage', 'cracks_found')),
    CONSTRAINT valid_job_aggregation CHECK (aggregation_level IN ('realtime', 'daily', 'weekly'))
);

-- Create agent_hashlists table to track hashlist distribution
CREATE TABLE IF NOT EXISTS agent_hashlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,  -- Changed from UUID to INTEGER
    hashlist_id BIGINT NOT NULL REFERENCES hashlists(id) ON DELETE CASCADE,  -- Changed from UUID to BIGINT
    file_path TEXT NOT NULL,
    downloaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    file_hash VARCHAR(32), -- MD5 hash for verification
    UNIQUE(agent_id, hashlist_id)
);

-- Create indexes for performance
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_priority ON job_executions(priority, created_at);
CREATE INDEX idx_job_tasks_agent_status ON job_tasks(agent_id, status);
CREATE INDEX idx_job_tasks_execution ON job_tasks(job_execution_id);
CREATE INDEX idx_agent_benchmarks_lookup ON agent_benchmarks(agent_id, attack_mode, hash_type);
CREATE INDEX idx_agent_metrics_lookup ON agent_performance_metrics(agent_id, metric_type, timestamp);
CREATE INDEX idx_agent_metrics_aggregation ON agent_performance_metrics(aggregation_level, timestamp);
CREATE INDEX idx_job_metrics_lookup ON job_performance_metrics(job_execution_id, metric_type, timestamp);
CREATE INDEX idx_agent_hashlists_cleanup ON agent_hashlists(last_used_at);