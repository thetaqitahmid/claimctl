-- Create health check configuration table
CREATE TABLE IF NOT EXISTS claimctl.resource_health_configs (
    resource_id UUID PRIMARY KEY REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT false,
    check_type VARCHAR(20) NOT NULL CHECK (check_type IN ('ping', 'http', 'tcp')),
    target VARCHAR(500) NOT NULL,
    interval_seconds INTEGER DEFAULT 60 CHECK (interval_seconds >= 10),
    timeout_seconds INTEGER DEFAULT 5 CHECK (timeout_seconds >= 1 AND timeout_seconds <= 30),
    retry_count INTEGER DEFAULT 3 CHECK (retry_count >= 0 AND retry_count <= 10),
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

-- Create health check status tracking table
CREATE TABLE IF NOT EXISTS claimctl.resource_health_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('healthy', 'degraded', 'down', 'unknown')),
    response_time_ms INTEGER,
    error_message TEXT,
    checked_at BIGINT NOT NULL,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

-- Create indexes for performance
CREATE INDEX idx_health_configs_enabled ON claimctl.resource_health_configs(enabled) WHERE enabled = true;
CREATE INDEX idx_health_status_resource_id ON claimctl.resource_health_status(resource_id);
CREATE INDEX idx_health_status_checked_at ON claimctl.resource_health_status(checked_at DESC);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER trigger_update_health_config_timestamp
    BEFORE UPDATE ON claimctl.resource_health_configs
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();
