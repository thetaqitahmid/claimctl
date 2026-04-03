CREATE TABLE IF NOT EXISTS claimctl.reservation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    reservation_id UUID REFERENCES claimctl.reservations(id) ON DELETE SET NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('created', 'activated', 'completed', 'cancelled', 'queue_position_changed')),
    user_id UUID NOT NULL REFERENCES claimctl.users(id),
    timestamp BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    details JSONB DEFAULT '{}'::jsonb
);

-- Indexes for performance
CREATE INDEX idx_reservation_history_resource ON claimctl.reservation_history(resource_id, timestamp DESC);
CREATE INDEX idx_reservation_history_reservation ON claimctl.reservation_history(reservation_id, timestamp DESC);
CREATE INDEX idx_reservation_history_user ON claimctl.reservation_history(user_id, timestamp DESC);
CREATE INDEX idx_reservation_history_action ON claimctl.reservation_history(action, timestamp DESC);

-- Trigger to automatically record reservation status changes
CREATE OR REPLACE FUNCTION claimctl.log_reservation_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Record the status change
    INSERT INTO claimctl.reservation_history (resource_id, reservation_id, action, user_id, details)
    VALUES (
        NEW.resource_id,
        NEW.id,
        CASE
            WHEN OLD.status IS NULL THEN 'created'
            WHEN NEW.status = 'active' AND OLD.status = 'pending' THEN 'activated'
            WHEN NEW.status = 'completed' AND OLD.status = 'active' THEN 'completed'
            WHEN NEW.status = 'cancelled' AND OLD.status IN ('pending', 'active') THEN 'cancelled'
            WHEN NEW.status = 'pending' AND OLD.status = 'active' THEN 'queue_position_changed'
            ELSE 'status_changed'
        END,
        NEW.user_id,
        jsonb_build_object(
            'old_status', OLD.status,
            'new_status', NEW.status,
            'old_queue_position', OLD.queue_position,
            'new_queue_position', NEW.queue_position,
            'start_time', NEW.start_time,
            'end_time', NEW.end_time
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic history logging
CREATE TRIGGER trigger_log_reservation_change
    AFTER UPDATE ON claimctl.reservations
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status OR OLD.queue_position IS DISTINCT FROM NEW.queue_position)
    EXECUTE FUNCTION claimctl.log_reservation_change();

-- Trigger for creation
CREATE TRIGGER trigger_log_reservation_creation
    AFTER INSERT ON claimctl.reservations
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.log_reservation_change();