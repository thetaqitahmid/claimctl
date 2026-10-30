ALTER TABLE claimctl.reservations
    ADD COLUMN lease_duration INTERVAL,
    ADD COLUMN lease_expires_at TIMESTAMPTZ,
    ADD COLUMN hard_end_time TIMESTAMPTZ;
ALTER TABLE claimctl.reservations ADD CONSTRAINT valid_lease
    CHECK ((lease_duration IS NULL AND lease_expires_at IS NULL)
        OR (lease_duration > INTERVAL '0 seconds' AND lease_expires_at IS NOT NULL));
CREATE TABLE claimctl.bundle_idempotency (
    user_id UUID NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    result JSONB NOT NULL,
    PRIMARY KEY (user_id, key)
);
ALTER TABLE claimctl.reservation_history ALTER COLUMN action TYPE VARCHAR(32);
CREATE OR REPLACE FUNCTION claimctl.log_reservation_change()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO claimctl.reservation_history
        (resource_id, reservation_id, action, user_id, details)
    VALUES (NEW.resource_id, NEW.id,
        CASE
            WHEN OLD.status IS NULL THEN 'created'
            WHEN NEW.status = 'active' AND OLD.status = 'pending' THEN 'activated'
            WHEN NEW.status = 'completed' AND OLD.status = 'active' THEN 'completed'
            WHEN NEW.status = 'cancelled' THEN 'cancelled'
            ELSE 'queue_position_changed'
        END,
        NEW.user_id,
        jsonb_build_object('old_status', OLD.status, 'new_status', NEW.status,
            'old_queue_position', OLD.queue_position,
            'new_queue_position', NEW.queue_position,
            'start_time', NEW.start_time, 'end_time', NEW.end_time));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
