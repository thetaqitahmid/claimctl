-- name: UpsertPreference :one
INSERT INTO claimctl.user_notification_preferences (user_id, event_type, channel, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, event_type, channel)
DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = extract(epoch from now())
RETURNING *;

-- name: GetUserPreferences :many
SELECT * FROM claimctl.user_notification_preferences
WHERE user_id = $1;

-- name: GetUserPreference :one
SELECT * FROM claimctl.user_notification_preferences
WHERE user_id = $1 AND event_type = $2 AND channel = $3;

