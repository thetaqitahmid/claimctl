-- name: CreateSecret :one
INSERT INTO claimctl.secrets (
    key, value, description
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetSecret :one
SELECT * FROM claimctl.secrets
WHERE id = $1 LIMIT 1;

-- name: GetSecretByKey :one
SELECT * FROM claimctl.secrets
WHERE key = $1 LIMIT 1;

-- name: ListSecrets :many
SELECT * FROM claimctl.secrets
ORDER BY key;

-- name: UpdateSecret :one
UPDATE claimctl.secrets
SET value = $2,
    description = $3,
    updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
WHERE id = $1
RETURNING *;

-- name: DeleteSecret :exec
DELETE FROM claimctl.secrets
WHERE id = $1;
