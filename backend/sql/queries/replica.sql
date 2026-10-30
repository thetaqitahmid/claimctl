-- name: LockRealtimeMutation :exec
SELECT pg_advisory_xact_lock(725025);

-- name: LockJWTInitialization :exec
SELECT pg_advisory_xact_lock(725026);

-- name: RealtimeBounds :one
SELECT COALESCE(min(id), 0)::bigint AS oldest, COALESCE(max(id), 0)::bigint AS newest
FROM claimctl.realtime_events;

-- name: ListRealtimeEvents :many
SELECT * FROM claimctl.realtime_events WHERE id > $1 ORDER BY id LIMIT 100;

-- name: GetSharedValue :one
SELECT value FROM claimctl.shared_storage WHERE key = $1 AND expires_at > clock_timestamp();

-- name: SetSharedValue :exec
INSERT INTO claimctl.shared_storage(key, value, expires_at)
VALUES ($1, $2, clock_timestamp() + sqlc.arg(ttl_seconds)::double precision * interval '1 second')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at;

-- name: DeleteSharedValue :exec
DELETE FROM claimctl.shared_storage WHERE key = $1;

-- name: ResetSharedValues :exec
DELETE FROM claimctl.shared_storage;

-- name: IncrementRateLimit :one
INSERT INTO claimctl.rate_limits(key, count, expires_at)
VALUES ($1, 1, clock_timestamp() + sqlc.arg(ttl_seconds)::double precision * interval '1 second')
ON CONFLICT (key) DO UPDATE SET
count = CASE WHEN claimctl.rate_limits.expires_at <= clock_timestamp() THEN 1 ELSE claimctl.rate_limits.count + 1 END,
expires_at = CASE WHEN claimctl.rate_limits.expires_at <= clock_timestamp() THEN clock_timestamp() + sqlc.arg(ttl_seconds)::double precision * interval '1 second' ELSE claimctl.rate_limits.expires_at END
RETURNING count, expires_at;

-- name: CleanupReplicaState :exec
DELETE FROM claimctl.realtime_events WHERE id IN (
 SELECT id FROM claimctl.realtime_events WHERE created_at < clock_timestamp() - interval '1 hour'
 OR id < (SELECT COALESCE(max(id), 0) - 10000 FROM claimctl.realtime_events)
 ORDER BY id LIMIT 1000
);

-- name: CleanupSharedValues :exec
DELETE FROM claimctl.shared_storage WHERE key IN (
 SELECT key FROM claimctl.shared_storage WHERE expires_at <= clock_timestamp() LIMIT 1000
);

-- name: CleanupRateLimits :exec
DELETE FROM claimctl.rate_limits WHERE key IN (
 SELECT key FROM claimctl.rate_limits WHERE expires_at <= clock_timestamp() LIMIT 1000
);
