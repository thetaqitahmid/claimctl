-- Add maintenance flag to resources
ALTER TABLE claimctl.resources
ADD COLUMN IF NOT EXISTS is_under_maintenance BOOLEAN DEFAULT false;

-- Create audit log table for maintenance changes
CREATE TABLE IF NOT EXISTS claimctl.maintenance_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES claimctl.resources(id) ON DELETE CASCADE,
    previous_state BOOLEAN NOT NULL,
    new_state BOOLEAN NOT NULL,
    changed_by UUID NOT NULL REFERENCES claimctl.users(id),
    changed_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    reason TEXT
);

-- Create index for efficient lookups by resource
CREATE INDEX IF NOT EXISTS idx_maintenance_audit_resource_id ON claimctl.maintenance_audit_log(resource_id);

-- Create index for efficient lookups by timestamp
CREATE INDEX IF NOT EXISTS idx_maintenance_audit_changed_at ON claimctl.maintenance_audit_log(changed_at DESC);
