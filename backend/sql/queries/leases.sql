-- name: SetReservationLifetime :one
UPDATE claimctl.reservations SET
    hard_end_time = to_timestamp(created_at) + sqlc.arg(max_duration)::interval,
    scheduled_end_time = LEAST(scheduled_end_time, to_timestamp(created_at) + sqlc.arg(max_duration)::interval),
    lease_duration = sqlc.narg(lease_duration)::interval,
    lease_expires_at = CASE WHEN sqlc.narg(lease_duration)::interval IS NOT NULL
        THEN LEAST(statement_timestamp() + sqlc.narg(lease_duration)::interval,
            to_timestamp(created_at) + sqlc.arg(max_duration)::interval, scheduled_end_time) ELSE NULL END
WHERE id = sqlc.arg(id) RETURNING *;

-- name: RenewReservationLease :one
UPDATE claimctl.reservations SET
    lease_duration = COALESCE(sqlc.narg(lease_duration)::interval, lease_duration),
    lease_expires_at = LEAST(statement_timestamp() + COALESCE(sqlc.narg(lease_duration)::interval, lease_duration), hard_end_time, scheduled_end_time),
    updated_at = FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::bigint
WHERE id = sqlc.arg(id) AND status IN ('active', 'pending')
    AND lease_expires_at > statement_timestamp()
    AND (hard_end_time IS NULL OR hard_end_time > statement_timestamp())
    AND (scheduled_end_time IS NULL OR scheduled_end_time > statement_timestamp())
RETURNING *;

-- name: ReclaimResourceReservations :many
UPDATE claimctl.reservations SET
    status = CASE WHEN status = 'pending' THEN 'cancelled' ELSE 'completed' END,
    end_time = GREATEST(start_time, FLOOR(EXTRACT(EPOCH FROM LEAST(statement_timestamp(), lease_expires_at, hard_end_time, scheduled_end_time)))::bigint),
    queue_position = NULL,
    updated_at = FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::bigint
WHERE resource_id = $1 AND status IN ('active', 'pending') AND
    (lease_expires_at <= statement_timestamp() OR hard_end_time <= statement_timestamp()
        OR scheduled_end_time <= statement_timestamp()) RETURNING *;

-- name: GetBundleIdempotency :one
SELECT * FROM claimctl.bundle_idempotency WHERE user_id=$1 AND key=$2;

-- name: SaveBundleIdempotency :exec
INSERT INTO claimctl.bundle_idempotency(user_id,key,fingerprint,result) VALUES($1,$2,$3,$4);

-- name: SetPendingQueuePosition :exec
UPDATE claimctl.reservations SET queue_position=$2
WHERE id=$1 AND status='pending';

-- name: FindMetricReservations :many
SELECT * FROM claimctl.reservations WHERE resource_id=$1;
