CREATE TABLE IF NOT EXISTS claimctl.groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS claimctl.group_members (
    group_id UUID REFERENCES claimctl.groups(id) ON DELETE CASCADE,
    user_id UUID REFERENCES claimctl.users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS claimctl.space_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id UUID REFERENCES claimctl.spaces(id) ON DELETE CASCADE,
    group_id UUID REFERENCES claimctl.groups(id) ON DELETE CASCADE,
    user_id UUID REFERENCES claimctl.users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK ((group_id IS NOT NULL AND user_id IS NULL) OR (group_id IS NULL AND user_id IS NOT NULL)),
    UNIQUE (space_id, group_id),
    UNIQUE (space_id, user_id)
);

-- Trigger for updating last modified timestamp
CREATE TRIGGER trigger_update_groups_last_modified
    BEFORE UPDATE ON claimctl.groups
    FOR EACH ROW
    EXECUTE FUNCTION claimctl.update_last_modified();
