CREATE TABLE IF NOT EXISTS claimctl.api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ, -- Nullable for non-expiring tokens
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_api_tokens_user_id ON claimctl.api_tokens(user_id);
CREATE INDEX idx_api_tokens_token_hash ON claimctl.api_tokens(token_hash);
