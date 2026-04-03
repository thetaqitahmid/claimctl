-- Drop audit log table
DROP TABLE IF EXISTS claimctl.maintenance_audit_log;

-- Remove maintenance flag from resources
ALTER TABLE claimctl.resources
DROP COLUMN IF EXISTS is_under_maintenance;
