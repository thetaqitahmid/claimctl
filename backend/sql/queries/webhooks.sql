-- name: CreateWebhook :one
INSERT INTO claimctl.webhooks (
    name, url, method, headers, template, description, signing_secret
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetWebhook :one
SELECT * FROM claimctl.webhooks
WHERE id = $1 LIMIT 1;

-- name: ListWebhooks :many
SELECT * FROM claimctl.webhooks
ORDER BY name;

-- name: UpdateWebhook :one
UPDATE claimctl.webhooks
SET name = $2,
    url = $3,
    method = $4,
    headers = $5,
    template = $6,
    description = $7,
    updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
WHERE id = $1
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM claimctl.webhooks
WHERE id = $1;
