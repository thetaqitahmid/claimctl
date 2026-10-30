DROP TABLE claimctl.claim_idempotency;
ALTER TABLE claimctl.reservations
    ALTER COLUMN created_at SET DEFAULT
    EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT;
