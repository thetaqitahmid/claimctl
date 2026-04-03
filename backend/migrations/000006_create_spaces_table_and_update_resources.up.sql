CREATE TABLE IF NOT EXISTS claimctl.spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
);

-- Update trigger for spaces
CREATE TRIGGER trigger_update_spaces_last_modified
    BEFORE UPDATE ON claimctl.spaces
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();

-- Add Foreign Key with Cascade Delete
ALTER TABLE claimctl.resources
    ADD CONSTRAINT fk_resources_spaces FOREIGN KEY (space_id) REFERENCES claimctl.spaces(id) ON DELETE CASCADE;
