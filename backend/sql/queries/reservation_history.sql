-- name: GetResourceReservationHistory :many
SELECT
    rh.id,
    rh.resource_id,
    rh.reservation_id,
    rh.action,
    rh.user_id,
    u.name as user_name,
    rh.timestamp,
    rh.details,
    r.name as resource_name
FROM claimctl.reservation_history rh
JOIN claimctl.users u ON rh.user_id = u.id
JOIN claimctl.resources r ON rh.resource_id = r.id
WHERE rh.resource_id = $1
ORDER BY rh.timestamp DESC;

-- name: GetReservationHistory :many
SELECT
    rh.id,
    rh.action,
    rh.user_id,
    u.name as user_name,
    rh.timestamp,
    rh.details
FROM claimctl.reservation_history rh
JOIN claimctl.users u ON rh.user_id = u.id
WHERE rh.reservation_id = $1
ORDER BY rh.timestamp DESC;

-- name: GetUserReservationHistory :many
SELECT
    rh.id,
    rh.resource_id,
    r.name as resource_name,
    rh.reservation_id,
    rh.action,
    rh.timestamp,
    rh.details
FROM claimctl.reservation_history rh
JOIN claimctl.resources r ON rh.resource_id = r.id
WHERE rh.user_id = $1
ORDER BY rh.timestamp DESC;

-- name: AddReservationHistoryLog :one
INSERT INTO claimctl.reservation_history (resource_id, reservation_id, action, user_id, details)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRecentHistoryByAction :many
SELECT
    rh.id,
    rh.resource_id,
    r.name as resource_name,
    rh.reservation_id,
    rh.action,
    rh.user_id,
    u.name as user_name,
    rh.timestamp,
    rh.details
FROM claimctl.reservation_history rh
JOIN claimctl.resources r ON rh.resource_id = r.id
JOIN claimctl.users u ON rh.user_id = u.id
WHERE rh.action = $1
ORDER BY rh.timestamp DESC
LIMIT $2;

-- name: GetResourceStats :one
SELECT
    COUNT(*) as total_reservations,
    COUNT(CASE WHEN rh.action = 'completed' THEN 1 END) as completed_reservations,
    COUNT(CASE WHEN rh.action = 'cancelled' THEN 1 END) as cancelled_reservations,
    COUNT(CASE WHEN rh.action = 'activated' THEN 1 END) as activated_reservations
FROM claimctl.reservation_history rh
WHERE rh.resource_id = $1;