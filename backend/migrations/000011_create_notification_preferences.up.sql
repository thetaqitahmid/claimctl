CREATE TABLE IF NOT EXISTS claimctl.user_notification_preferences (
    user_id UUID NOT NULL REFERENCES claimctl.users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL, -- 'email', 'slack', 'teams'
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at BIGINT DEFAULT (extract(epoch from now())),
    updated_at BIGINT DEFAULT (extract(epoch from now())),
    PRIMARY KEY (user_id, event_type, channel)
);

-- Add channel configuration to users table
ALTER TABLE claimctl.users ADD COLUMN IF NOT EXISTS slack_destination TEXT;
ALTER TABLE claimctl.users ADD COLUMN IF NOT EXISTS teams_webhook_url TEXT;

