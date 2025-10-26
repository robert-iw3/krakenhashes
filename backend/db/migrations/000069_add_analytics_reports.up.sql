-- Create analytics_reports table for password analytics
CREATE TABLE analytics_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    analytics_data JSONB,
    total_hashlists INTEGER,
    total_hashes INTEGER,
    total_cracked INTEGER,
    queue_position INTEGER,
    custom_patterns TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    CONSTRAINT valid_report_status CHECK (
        status IN ('queued', 'processing', 'completed', 'failed')
    ),
    CONSTRAINT valid_date_range CHECK (end_date >= start_date)
);

-- Create indexes for efficient querying
CREATE INDEX idx_analytics_reports_client ON analytics_reports(client_id);
CREATE INDEX idx_analytics_reports_status ON analytics_reports(status);
CREATE INDEX idx_analytics_reports_dates ON analytics_reports(client_id, start_date, end_date);
CREATE INDEX idx_analytics_reports_queue ON analytics_reports(queue_position) WHERE status = 'queued';

-- Add comments for documentation
COMMENT ON TABLE analytics_reports IS 'Stores generated password analytics reports for client engagements';
COMMENT ON COLUMN analytics_reports.analytics_data IS 'JSONB containing all calculated analytics metrics';
COMMENT ON COLUMN analytics_reports.custom_patterns IS 'User-defined organization name variations to check';
COMMENT ON COLUMN analytics_reports.queue_position IS 'Position in processing queue (NULL when not queued)';
COMMENT ON COLUMN analytics_reports.status IS 'Report status: queued (waiting), processing (generating), completed (ready), failed (error occurred)';
