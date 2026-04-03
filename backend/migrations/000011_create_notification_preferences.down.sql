DROP TABLE IF EXISTS claimctl.user_notification_preferences;

ALTER TABLE claimctl.users DROP COLUMN IF EXISTS slack_destination;
ALTER TABLE claimctl.users DROP COLUMN IF EXISTS teams_webhook_url;

