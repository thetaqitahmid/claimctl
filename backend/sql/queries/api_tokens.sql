-- name: CreateAPIToken :one
INSERT INTO claimctl.api_tokens (user_id, name, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAPITokens :many
SELECT * FROM claimctl.api_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIToken :exec
DELETE FROM claimctl.api_tokens
WHERE id = $1 AND user_id = $2;

-- name: GetAPITokenByHash :one
SELECT * FROM claimctl.api_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: UpdateAPITokenLastUsed :exec
UPDATE claimctl.api_tokens
SET last_used_at = NOW()
WHERE id = $1;
