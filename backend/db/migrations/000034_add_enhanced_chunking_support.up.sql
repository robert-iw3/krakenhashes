-- Migration: Add Enhanced Chunking Support for Rule Multiplication and Combination Attacks
-- This migration adds columns to track keyspace multiplication factors and rule splitting

-- Add columns to job_executions table for keyspace tracking
ALTER TABLE job_executions 
ADD COLUMN base_keyspace BIGINT,           -- Wordlist-only keyspace
ADD COLUMN effective_keyspace BIGINT,       -- Base × multiplication factor
ADD COLUMN multiplication_factor INT DEFAULT 1,
ADD COLUMN uses_rule_splitting BOOLEAN DEFAULT FALSE,
ADD COLUMN rule_split_count INT DEFAULT 0;

-- Add index for performance on rule splitting queries
CREATE INDEX idx_job_executions_rule_splitting ON job_executions(uses_rule_splitting);

-- Add columns to job_tasks table for rule chunk tracking
ALTER TABLE job_tasks
ADD COLUMN rule_start_index INT,
ADD COLUMN rule_end_index INT,
ADD COLUMN rule_chunk_path TEXT,
ADD COLUMN is_rule_split_task BOOLEAN DEFAULT FALSE,
ADD COLUMN priority INT DEFAULT 0,
ADD COLUMN attack_cmd TEXT;

-- Add index for rule split tasks
CREATE INDEX idx_job_tasks_rule_split ON job_tasks(is_rule_split_task);

-- Insert system settings for rule splitting configuration
INSERT INTO system_settings (key, value, description) VALUES
('rule_split_enabled', 'true', 'Enable/disable rule splitting feature'),
('rule_split_threshold', '2.0', 'Multiplier for triggering rule split (job time > threshold × chunk time)'),
('rule_split_min_rules', '100', 'Minimum number of rules to consider splitting'),
('rule_split_max_chunks', '1000', 'Maximum number of rule chunks to create'),
('rule_chunk_temp_dir', '/data/krakenhashes/temp/rule_chunks', 'Directory for temporary rule chunk files')
ON CONFLICT (key) DO UPDATE 
SET value = EXCLUDED.value,
    description = EXCLUDED.description;