-- name: CreateReservation :one
INSERT INTO claimctl.reservations (resource_id, user_id, status, queue_position, start_time, duration, scheduled_end_time, updated_at)
VALUES ($1, $2,
    CASE
        WHEN (
            SELECT COUNT(*) FROM claimctl.reservations
            WHERE resource_id = $1 AND status IN ('pending', 'active')
        ) = 0 THEN 'active'
        ELSE 'pending'
    END,
    CASE
        WHEN (
            SELECT COUNT(*) FROM claimctl.reservations
            WHERE resource_id = $1 AND status IN ('pending', 'active')
        ) = 0 THEN 0
        ELSE COALESCE((
            SELECT MAX(queue_position) FROM claimctl.reservations
            WHERE resource_id = $1 AND status = 'pending'
        ), 0) + 1
    END,
    CASE
        WHEN (
            SELECT COUNT(*) FROM claimctl.reservations
            WHERE resource_id = $1 AND status IN ('pending', 'active')
        ) = 0 THEN FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
        ELSE NULL
    END,
    $3,
    CASE
        WHEN (
            SELECT COUNT(*) FROM claimctl.reservations
            WHERE resource_id = $1 AND status IN ('pending', 'active')
        ) = 0 AND $3::INTERVAL IS NOT NULL THEN NOW() + $3::INTERVAL
        ELSE NULL
    END,
    FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
)
RETURNING *;

-- name: FindReservationById :one
SELECT * FROM claimctl.reservations WHERE id=$1;

-- name: FindUserReservationForResource :one
SELECT * FROM claimctl.reservations
WHERE user_id=$1 AND resource_id=$2 AND status IN ('pending', 'active')
ORDER BY
    CASE status
        WHEN 'active' THEN 1
        WHEN 'pending' THEN 2
    END,
    queue_position
LIMIT 1;

-- name: FindActiveReservationByResource :one
SELECT * FROM claimctl.reservations
WHERE resource_id=$1 AND status='active';

-- name: FindPendingReservationsByResource :many
SELECT * FROM claimctl.reservations
WHERE resource_id=$1 AND status='pending'
ORDER BY queue_position;

-- name: FindUserActiveReservations :many
SELECT r.*, res.name as resource_name, res.type as resource_type
FROM claimctl.reservations r
JOIN claimctl.resources res ON r.resource_id = res.id
WHERE r.user_id=$1 AND r.status IN ('pending', 'active')
ORDER BY r.created_at;

-- name: ActivateReservation :execrows
UPDATE claimctl.reservations
SET
    status='active',
    start_time=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT,
    scheduled_end_time = LEAST(CASE WHEN duration IS NOT NULL THEN (statement_timestamp() + duration) ELSE NULL END, hard_end_time),
    queue_position=0,
    updated_at=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE claimctl.reservations.id=$1 AND claimctl.reservations.status='pending'
    AND (lease_expires_at IS NULL OR lease_expires_at > statement_timestamp())
    AND (hard_end_time IS NULL OR hard_end_time > statement_timestamp())
    AND NOT EXISTS (SELECT 1 FROM claimctl.reservations active WHERE active.resource_id=claimctl.reservations.resource_id AND active.status='active');

-- name: CompleteReservation :exec
UPDATE claimctl.reservations
SET status='completed', end_time=GREATEST(start_time, FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT), updated_at=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE id=$1 AND status='active';

-- name: CancelReservation :exec
UPDATE claimctl.reservations
SET status='cancelled', end_time=GREATEST(start_time, FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT), queue_position=NULL, updated_at=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE id=$1 AND status IN ('pending', 'active');

-- name: GetNextInQueue :one
SELECT * FROM claimctl.reservations
WHERE resource_id=$1 AND status='pending'
ORDER BY queue_position
LIMIT 1;

-- name: PromoteNextInQueue :exec
UPDATE claimctl.reservations r
SET queue_position = queue_position - 1, updated_at = FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE r.resource_id=$1 AND r.status='pending' AND r.queue_position > 1;

-- name: UpdateQueuePositions :exec
UPDATE claimctl.reservations r
SET queue_position = queue_position - 1, updated_at = FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE r.resource_id=$1 AND r.status='pending' AND r.queue_position > $2;

-- name: FindAllReservations :many
SELECT r.*, u.name as user_name, res.name as resource_name
FROM claimctl.reservations r
JOIN claimctl.users u ON r.user_id = u.id
JOIN claimctl.resources res ON r.resource_id = res.id
ORDER BY r.created_at DESC;

-- name: FindReservationsByResource :many
SELECT r.*, u.name as user_name
FROM claimctl.reservations r
JOIN claimctl.users u ON r.user_id = u.id
WHERE r.resource_id=$1
ORDER BY r.created_at DESC;

-- name: FindReservationsByUser :many
SELECT r.*, res.name as resource_name, res.type as resource_type
FROM claimctl.reservations r
JOIN claimctl.resources res ON r.resource_id = res.id
WHERE r.user_id=$1
ORDER BY r.created_at DESC;

-- name: CountPendingReservations :one
SELECT COUNT(*) as count
FROM claimctl.reservations
WHERE resource_id=$1 AND status='pending';

-- name: GetUserQueuePosition :one
SELECT queue_position
FROM claimctl.reservations
WHERE user_id=$1 AND resource_id=$2 AND status='pending';

-- name: UpdateReservationStatus :exec
UPDATE claimctl.reservations
SET status=$2, updated_at=$3
WHERE id=$1;

-- name: FindExpiredActiveReservations :many
SELECT * FROM claimctl.reservations
WHERE status IN ('active', 'pending') AND
    (scheduled_end_time <= statement_timestamp() OR lease_expires_at <= statement_timestamp()
        OR hard_end_time <= statement_timestamp())
ORDER BY resource_id, id LIMIT 1000;

-- name: ExpireReservation :exec
UPDATE claimctl.reservations
SET status='completed', end_time=GREATEST(start_time, FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT), updated_at=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE id=$1;

-- name: GetQueueForResource :many
SELECT r.id, r.user_id, r.status, r.queue_position, r.start_time, r.end_time, r.scheduled_end_time, r.created_at, r.duration, r.lease_expires_at, r.hard_end_time, u.name as user_name, u.email as user_email
FROM claimctl.reservations r
JOIN claimctl.users u ON r.user_id = u.id
WHERE r.resource_id=$1 AND r.status IN ('active', 'pending')
ORDER BY
    CASE r.status
        WHEN 'active' THEN 1
        WHEN 'pending' THEN 2
    END,
    r.queue_position;

-- name: CancelAllReservationsForResource :exec
UPDATE claimctl.reservations
SET status='cancelled', end_time=GREATEST(start_time, FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT), queue_position=NULL, updated_at=FLOOR(EXTRACT(EPOCH FROM statement_timestamp()))::BIGINT
WHERE resource_id=$1 AND status IN ('pending', 'active');