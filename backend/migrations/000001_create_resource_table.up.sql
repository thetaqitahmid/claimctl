CREATE SCHEMA IF NOT EXISTS claimctl;

CREATE TABLE IF NOT EXISTS claimctl.resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL,
    labels JSONB,
    properties JSONB,
    space_id UUID NOT NULL,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

CREATE INDEX idx_resources_name ON claimctl.resources(name);

CREATE OR REPLACE FUNCTION claimctl.update_last_modified()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_last_modified
    BEFORE UPDATE ON claimctl.resources
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();
