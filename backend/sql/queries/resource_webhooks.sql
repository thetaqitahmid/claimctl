-- name: AddResourceWebhook :exec
INSERT INTO claimctl.resource_webhooks (
    resource_id, webhook_id, events
) VALUES (
    $1, $2, $3
) ON CONFLICT (resource_id, webhook_id) DO UPDATE
SET events = $3;

-- name: RemoveResourceWebhook :exec
DELETE FROM claimctl.resource_webhooks
WHERE resource_id = $1 AND webhook_id = $2;

-- name: GetResourceWebhooks :many
SELECT w.*, rw.events
FROM claimctl.webhooks w
JOIN claimctl.resource_webhooks rw ON w.id = rw.webhook_id
WHERE rw.resource_id = $1;

-- name: GetWebhooksForEvent :many
SELECT w.*
FROM claimctl.webhooks w
JOIN claimctl.resource_webhooks rw ON w.id = rw.webhook_id
WHERE rw.resource_id = $1 AND $2::text = ANY(rw.events);
