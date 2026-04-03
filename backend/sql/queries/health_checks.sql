-- name: GetHealthConfig :one
SELECT * FROM claimctl.resource_health_configs WHERE resource_id = $1;

-- name: UpsertHealthConfig :one
INSERT INTO claimctl.resource_health_configs (
    resource_id, enabled, check_type, target, interval_seconds, timeout_seconds, retry_count
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (resource_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    check_type = EXCLUDED.check_type,
    target = EXCLUDED.target,
    interval_seconds = EXCLUDED.interval_seconds,
    timeout_seconds = EXCLUDED.timeout_seconds,
    retry_count = EXCLUDED.retry_count,
    updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
RETURNING *;

-- name: DeleteHealthConfig :exec
DELETE FROM claimctl.resource_health_configs WHERE resource_id = $1;

-- name: GetHealthStatus :one
SELECT * FROM claimctl.resource_health_status
WHERE resource_id = $1
ORDER BY checked_at DESC
LIMIT 1;

-- name: GetHealthHistory :many
SELECT * FROM claimctl.resource_health_status
WHERE resource_id = $1
ORDER BY checked_at DESC
LIMIT $2;

-- name: CreateHealthStatus :one
INSERT INTO claimctl.resource_health_status (
    resource_id, status, response_time_ms, error_message, checked_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAllEnabledHealthConfigs :many
SELECT * FROM claimctl.resource_health_configs
WHERE enabled = true;

-- name: GetResourcesDueForCheck :many
SELECT
    c.resource_id,
    c.interval_seconds,
    COALESCE(MAX(s.checked_at), 0) as last_checked_at
FROM claimctl.resource_health_configs c
LEFT JOIN claimctl.resource_health_status s ON c.resource_id = s.resource_id
WHERE c.enabled = true
GROUP BY c.resource_id, c.interval_seconds
HAVING COALESCE(MAX(s.checked_at), 0) + c.interval_seconds <= EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT;

-- name: CleanupOldHealthStatus :exec
DELETE FROM claimctl.resource_health_status
WHERE checked_at < $1;
