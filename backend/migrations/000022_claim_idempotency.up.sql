CREATE TABLE claimctl.claim_idempotency (
    user_id UUID NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    reservation_id UUID NOT NULL REFERENCES claimctl.reservations(id) ON DELETE CASCADE,
    result JSONB NOT NULL,
    PRIMARY KEY (user_id, key)
);
ALTER TABLE claimctl.reservations
    ALTER COLUMN created_at SET DEFAULT
    FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT;
