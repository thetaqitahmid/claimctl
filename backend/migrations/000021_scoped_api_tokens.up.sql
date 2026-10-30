ALTER TABLE claimctl.api_tokens
    ADD COLUMN scopes TEXT[] NOT NULL DEFAULT ARRAY['read', 'reserve'],
    ADD COLUMN space_ids UUID[] NOT NULL DEFAULT '{}';

UPDATE claimctl.api_tokens
SET expires_at = LEAST(COALESCE(expires_at, NOW() + INTERVAL '30 days'),
                       NOW() + INTERVAL '90 days');
