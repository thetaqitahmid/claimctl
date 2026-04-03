CREATE TABLE IF NOT EXISTS claimctl.secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

CREATE TRIGGER trigger_update_secrets_last_modified
    BEFORE UPDATE ON claimctl.secrets
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();

CREATE TABLE IF NOT EXISTS claimctl.webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB,
    template TEXT,
    description TEXT,
    signing_secret TEXT NOT NULL DEFAULT md5(random()::text),
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

CREATE TRIGGER trigger_update_webhooks_last_modified
    BEFORE UPDATE ON claimctl.webhooks
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();

CREATE TABLE IF NOT EXISTS claimctl.resource_webhooks (
    resource_id UUID NOT NULL REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    webhook_id UUID NOT NULL REFERENCES claimctl.webhooks(id) ON DELETE CASCADE,
    events TEXT[] NOT NULL,
    PRIMARY KEY (resource_id, webhook_id)
);
