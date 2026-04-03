-- name: FindAllResources :many
SELECT * FROM claimctl.resources;

-- name: FindResourceById :one
SELECT * FROM claimctl.resources WHERE id=$1;

-- name: FindResourceByName :one
SELECT * FROM claimctl.resources WHERE name=$1;

-- name: CreateNewResource :one
INSERT INTO claimctl.resources (name, labels, type, space_id, properties)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteResourceById :exec
DELETE FROM claimctl.resources WHERE id=$1;

-- name: UpdateResourceById :exec
UPDATE claimctl.resources SET name=$1, labels=$2, type=$3, properties=$4
WHERE id=$5;

-- name: VerifyResourceNameIsUnique :one
SELECT COUNT(*) FROM claimctl.resources WHERE name=$1;

-- name: GetResourceTest :one
SELECT id, name FROM claimctl.resources WHERE id=$1;

-- name: GetResourceReservationStatus :one
SELECT
    r.id,
    r.name,
    r.type,
    r.labels,
    r.space_id,
    r.created_at,
    r.updated_at,
    r.properties,
    r.is_under_maintenance,
    COALESCE(active_reservations.count, 0) as active_reservations,
    COALESCE(pending_reservations.count, 0) as queue_length,
    COALESCE(next_reservation.user_id, '00000000-0000-0000-0000-000000000000'::uuid) as next_user_id,
    COALESCE(next_reservation.queue_position, 0) as next_queue_position,
    active_res_details.start_time as active_reservation_start_time,
    active_res_details.duration as active_reservation_duration,
    active_res_details.created_at as active_reservation_created_at
FROM claimctl.resources r
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'active'
    GROUP BY resource_id
) active_reservations ON r.id = active_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'pending'
    GROUP BY resource_id
) pending_reservations ON r.id = pending_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, user_id, queue_position
    FROM claimctl.reservations
    WHERE status = 'pending' AND queue_position = 1
) next_reservation ON r.id = next_reservation.resource_id
LEFT JOIN claimctl.reservations active_res_details
  ON r.id = active_res_details.resource_id AND active_res_details.status = 'active'
WHERE r.id = $1;

-- name: GetAllResourcesWithReservationStatus :many
SELECT
    r.id,
    r.name,
    r.type,
    r.labels,
    r.space_id,
    r.created_at,
    r.updated_at,
    r.properties,
    r.is_under_maintenance,
    COALESCE(active_reservations.count, 0) as active_reservations,
    COALESCE(pending_reservations.count, 0) as queue_length,
    COALESCE(next_reservation.user_id, '00000000-0000-0000-0000-000000000000'::uuid) as next_user_id,
    COALESCE(next_reservation.queue_position, 0) as next_queue_position,
    active_res_details.start_time as active_reservation_start_time,
    active_res_details.duration as active_reservation_duration,
    active_res_details.created_at as active_reservation_created_at
FROM claimctl.resources r
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'active'
    GROUP BY resource_id
) active_reservations ON r.id = active_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'pending'
    GROUP BY resource_id
) pending_reservations ON r.id = pending_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, user_id, queue_position
    FROM claimctl.reservations
    WHERE status = 'pending' AND queue_position = 1
) next_reservation ON r.id = next_reservation.resource_id
LEFT JOIN claimctl.reservations active_res_details
  ON r.id = active_res_details.resource_id AND active_res_details.status = 'active';

-- name: GetAllResourcesWithReservationStatusForUser :many
SELECT DISTINCT
    r.id,
    r.name,
    r.type,
    r.labels,
    r.space_id,
    r.created_at,
    r.updated_at,
    r.properties,
    r.is_under_maintenance,
    COALESCE(active_reservations.count, 0) as active_reservations,
    COALESCE(pending_reservations.count, 0) as queue_length,
    COALESCE(next_reservation.user_id, '00000000-0000-0000-0000-000000000000'::uuid) as next_user_id,
    COALESCE(next_reservation.queue_position, 0) as next_queue_position,
    active_res_details.start_time as active_reservation_start_time,
    active_res_details.duration as active_reservation_duration,
    active_res_details.created_at as active_reservation_created_at
FROM claimctl.resources r
JOIN claimctl.spaces s ON r.space_id = s.id
LEFT JOIN claimctl.space_permissions sp ON s.id = sp.space_id
LEFT JOIN claimctl.group_members gm ON sp.group_id = gm.group_id
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'active'
    GROUP BY resource_id
) active_reservations ON r.id = active_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, COUNT(*) as count
    FROM claimctl.reservations
    WHERE status = 'pending'
    GROUP BY resource_id
) pending_reservations ON r.id = pending_reservations.resource_id
LEFT JOIN (
    SELECT resource_id, user_id, queue_position
    FROM claimctl.reservations
    WHERE status = 'pending' AND queue_position = 1
) next_reservation ON r.id = next_reservation.resource_id
LEFT JOIN claimctl.reservations active_res_details
  ON r.id = active_res_details.resource_id AND active_res_details.status = 'active'
WHERE
    s.name = 'Default Space' OR
    sp.user_id = $1 OR
    (sp.group_id IS NOT NULL AND gm.user_id = $1)
ORDER BY r.id;

-- name: SetResourceMaintenanceMode :one
UPDATE claimctl.resources
SET is_under_maintenance = $2, updated_at = $3
WHERE id = $1
RETURNING *;

-- name: GetResourceMaintenanceStatus :one
SELECT is_under_maintenance FROM claimctl.resources WHERE id = $1;

-- name: GetResourceName :one
SELECT name FROM claimctl.resources WHERE id = $1;

-- name: LogMaintenanceChange :one
INSERT INTO claimctl.maintenance_audit_log (resource_id, previous_state, new_state, changed_by, reason)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMaintenanceHistory :many
SELECT
    mal.*,
    u.email as changed_by_email
FROM claimctl.maintenance_audit_log mal
JOIN claimctl.users u ON mal.changed_by = u.id
WHERE mal.resource_id = $1
ORDER BY mal.changed_at DESC;

-- name: AcquireResourceLock :one
SELECT id FROM claimctl.resources WHERE id=$1 FOR UPDATE;