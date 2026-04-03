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
        ) = 0 THEN EXTRACT(EPOCH FROM NOW())::BIGINT
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
    EXTRACT(EPOCH FROM NOW())::BIGINT
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

-- name: ActivateReservation :exec
UPDATE claimctl.reservations
SET
    status='active',
    start_time=$2,
    scheduled_end_time = CASE WHEN duration IS NOT NULL THEN (to_timestamp($3) + duration) ELSE NULL END,
    queue_position=0,
    updated_at=$4
WHERE id=$1 AND status='pending';

-- name: CompleteReservation :exec
UPDATE claimctl.reservations
SET status='completed', end_time=$2, updated_at=$3
WHERE id=$1 AND status='active';

-- name: CancelReservation :exec
UPDATE claimctl.reservations
SET status='cancelled', end_time=$2, queue_position=NULL, updated_at=$3
WHERE id=$1 AND status IN ('pending', 'active');

-- name: GetNextInQueue :one
SELECT * FROM claimctl.reservations
WHERE resource_id=$1 AND status='pending'
ORDER BY queue_position
LIMIT 1;

-- name: PromoteNextInQueue :exec
UPDATE claimctl.reservations r
SET queue_position = queue_position - 1, updated_at = $2
WHERE r.resource_id=$1 AND r.status='pending' AND r.queue_position > 1;

-- name: UpdateQueuePositions :exec
UPDATE claimctl.reservations r
SET queue_position = queue_position - 1, updated_at = $2
WHERE r.resource_id=$1 AND r.status='pending' AND r.queue_position > $3;

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
WHERE status='active' AND scheduled_end_time IS NOT NULL AND scheduled_end_time < NOW();

-- name: ExpireReservation :exec
UPDATE claimctl.reservations
SET status='completed', end_time=EXTRACT(EPOCH FROM NOW())::BIGINT, updated_at=$2
WHERE id=$1;

-- name: GetQueueForResource :many
SELECT r.id, r.user_id, r.status, r.queue_position, r.start_time, r.end_time, r.scheduled_end_time, r.created_at, r.duration, u.name as user_name, u.email as user_email
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
SET status='cancelled', end_time=$2, queue_position=NULL, updated_at=$2
WHERE resource_id=$1 AND status IN ('pending', 'active');