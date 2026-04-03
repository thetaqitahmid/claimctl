CREATE TABLE IF NOT EXISTS claimctl.reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed', 'cancelled')),
    queue_position INTEGER,
    start_time BIGINT,
    end_time BIGINT,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    scheduled_end_time TIMESTAMPTZ,
    duration INTERVAL
);

-- Indexes for performance
CREATE INDEX idx_reservations_resource_status ON claimctl.reservations(resource_id, status, created_at);
CREATE INDEX idx_reservations_user_active ON claimctl.reservations(user_id) WHERE status IN ('pending', 'active');
CREATE INDEX idx_reservations_status_time ON claimctl.reservations(status, start_time, end_time);
CREATE INDEX idx_reservations_queue_order ON claimctl.reservations(resource_id, status, queue_position) WHERE status = 'pending';
CREATE INDEX idx_reservations_scheduled_end_time ON claimctl.reservations(scheduled_end_time) WHERE scheduled_end_time IS NOT NULL;

-- Unique constraint: Only one active reservation per resource
CREATE UNIQUE INDEX idx_resource_active_reservation
ON claimctl.reservations (resource_id)
WHERE status = 'active';

-- Additional constraints for data integrity
-- Prevent user from having multiple active reservations for the same resource
CREATE UNIQUE INDEX idx_user_resource_active
ON claimctl.reservations (user_id, resource_id)
WHERE status IN ('pending', 'active');

-- Ensure time fields are set appropriately for status
ALTER TABLE claimctl.reservations ADD CONSTRAINT check_time_fields
CHECK (
    (status = 'pending' AND start_time IS NULL AND end_time IS NULL) OR
    (status = 'active' AND start_time IS NOT NULL AND end_time IS NULL) OR
    (status = 'completed' AND start_time IS NOT NULL AND end_time IS NOT NULL) OR
    (status = 'cancelled' AND end_time IS NOT NULL)
);

-- Ensure end_time is after start_time when both are set
ALTER TABLE claimctl.reservations ADD CONSTRAINT check_time_order
CHECK (
    (start_time IS NULL OR end_time IS NULL) OR
    (end_time >= start_time)
);

-- Queue position constraints
ALTER TABLE claimctl.reservations ADD CONSTRAINT check_queue_position
CHECK (
    (status IN ('pending') AND queue_position > 0) OR
    (status IN ('active', 'completed', 'cancelled') AND (queue_position IS NULL OR queue_position = 0))
);

-- Ensure unique queue positions per resource for pending reservations
CREATE UNIQUE INDEX idx_resource_queue_position
ON claimctl.reservations (resource_id, queue_position)
WHERE status = 'pending';

-- Update trigger for epoch time
CREATE TRIGGER trigger_update_reservations_last_modified
    BEFORE UPDATE ON claimctl.reservations
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();