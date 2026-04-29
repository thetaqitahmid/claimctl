CREATE TABLE claimctl.refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    family_id  UUID        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id   ON claimctl.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family_id ON claimctl.refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_expires_at ON claimctl.refresh_tokens(expires_at);
