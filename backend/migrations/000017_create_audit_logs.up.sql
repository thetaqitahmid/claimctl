CREATE TABLE claimctl.audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES claimctl.users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255),
    changes JSONB,
    ip_address VARCHAR(45),
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_audit_logs_actor ON claimctl.audit_logs(actor_id);
CREATE INDEX idx_audit_logs_entity_type_id ON claimctl.audit_logs(entity_type, entity_id);
