CREATE TABLE claimctl.webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES claimctl.webhooks(id) ON DELETE CASCADE,
    event VARCHAR(255) NOT NULL,
    status_code INT NOT NULL,
    request_body TEXT NOT NULL,
    response_body TEXT NOT NULL,
    duration_ms INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_logs_webhook_id ON claimctl.webhook_logs(webhook_id);
CREATE INDEX idx_webhook_logs_created_at ON claimctl.webhook_logs(created_at);
