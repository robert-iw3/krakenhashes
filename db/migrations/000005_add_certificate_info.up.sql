ALTER TABLE agents
    ADD COLUMN certificate_info JSONB NOT NULL DEFAULT '{}'; 