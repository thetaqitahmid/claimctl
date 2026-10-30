-- name: LockClaimUser :one
SELECT * FROM claimctl.users WHERE id=$1 FOR NO KEY UPDATE;

-- name: CountLiveClaims :one
SELECT COUNT(*) FROM claimctl.reservations
WHERE user_id=$1 AND status IN ('active', 'pending')
  AND (lease_expires_at IS NULL OR lease_expires_at > statement_timestamp())
  AND (hard_end_time IS NULL OR hard_end_time > statement_timestamp())
  AND (scheduled_end_time IS NULL OR scheduled_end_time > statement_timestamp());

-- name: FindClaimCandidates :many
SELECT * FROM claimctl.resources ORDER BY id;

-- name: GetClaimIdempotency :one
SELECT * FROM claimctl.claim_idempotency WHERE user_id=$1 AND key=$2;

-- name: SaveClaimIdempotency :exec
INSERT INTO claimctl.claim_idempotency
(user_id, key, fingerprint, reservation_id, result) VALUES ($1,$2,$3,$4,$5);
