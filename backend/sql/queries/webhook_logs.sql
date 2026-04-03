-- name: CreateWebhookLog :one
INSERT INTO claimctl.webhook_logs (
  webhook_id, event, status_code, request_body, response_body, duration_ms
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetWebhookLogs :many
SELECT * FROM claimctl.webhook_logs
WHERE webhook_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CleanupOldWebhookLogs :exec
DELETE FROM claimctl.webhook_logs
WHERE created_at < $1;
