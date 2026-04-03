-- name: UpdateUserChannelConfig :one
UPDATE claimctl.users
SET slack_destination = $2, teams_webhook_url = $3, notification_email = $4
WHERE id = $1
RETURNING *;
