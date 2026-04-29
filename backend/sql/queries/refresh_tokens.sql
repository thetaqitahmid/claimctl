-- name: CreateRefreshToken :one
INSERT INTO claimctl.refresh_tokens (user_id, token_hash, family_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM claimctl.refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshTokenFamily :exec
UPDATE claimctl.refresh_tokens
SET revoked = TRUE
WHERE family_id = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM claimctl.refresh_tokens
WHERE token_hash = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE claimctl.refresh_tokens
SET revoked = TRUE
WHERE user_id = $1 AND revoked = FALSE;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM claimctl.refresh_tokens
WHERE expires_at < NOW();
