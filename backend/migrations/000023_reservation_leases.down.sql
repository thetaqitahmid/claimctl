DROP TABLE claimctl.bundle_idempotency;
ALTER TABLE claimctl.reservations DROP CONSTRAINT valid_lease;
ALTER TABLE claimctl.reservations DROP COLUMN lease_duration,
    DROP COLUMN lease_expires_at, DROP COLUMN hard_end_time;
