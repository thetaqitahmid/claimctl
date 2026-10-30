-- name: ClaimWebhookDelivery :one
WITH candidate AS (
    SELECT id FROM claimctl.webhook_outbox
    WHERE (state = 'pending' AND available_at <= clock_timestamp())
       OR (state = 'delivering' AND lease_until <= clock_timestamp())
    ORDER BY available_at, created_at FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE claimctl.webhook_outbox o SET
    state = CASE WHEN attempts >= 8 THEN 'dead' ELSE 'delivering' END,
    fence = CASE WHEN attempts >= 8 THEN NULL::uuid ELSE sqlc.narg(fence)::uuid END,
    lease_until = CASE WHEN attempts >= 8 THEN NULL::timestamptz
        ELSE clock_timestamp() + INTERVAL '30 seconds' END,
    last_error = CASE WHEN attempts >= 8 THEN 'attempt_limit' ELSE last_error END,
    attempts = LEAST(attempts + 1, 8)
FROM candidate c WHERE o.id = c.id RETURNING o.*;

-- name: FinishWebhookDelivery :execrows
UPDATE claimctl.webhook_outbox SET state = sqlc.arg(state), last_status = sqlc.arg(last_status), last_error = sqlc.arg(last_error),
    available_at = clock_timestamp() + sqlc.arg(delay_seconds)::integer * INTERVAL '1 second',
    lease_until = NULL, fence = NULL
WHERE id = sqlc.arg(id) AND fence = sqlc.arg(fence) AND state = 'delivering'
    AND lease_until > clock_timestamp();

-- name: ListWebhookDeliveries :many
SELECT id, webhook_id, resource_id, event, state, attempts, available_at,
    lease_until, last_status, last_error, created_at FROM claimctl.webhook_outbox
WHERE webhook_id = $1 OR ($1::uuid = '00000000-0000-0000-0000-000000000000'::uuid AND state = 'dead')
ORDER BY created_at DESC LIMIT 100;

-- name: CleanupDeliveredWebhooks :exec
DELETE FROM claimctl.webhook_outbox
WHERE state = 'delivered' AND created_at < sqlc.arg(cutoff);

-- name: AcquireWorkerLease :one
INSERT INTO claimctl.worker_leases (name, fence, lease_until)
VALUES (sqlc.arg(name), sqlc.arg(fence), clock_timestamp() + INTERVAL '60 seconds')
ON CONFLICT (name) DO UPDATE SET fence = EXCLUDED.fence, lease_until = EXCLUDED.lease_until
WHERE worker_leases.lease_until <= clock_timestamp()
RETURNING fence;

-- name: FinishWorkerLease :execrows
UPDATE claimctl.worker_leases SET lease_until = clock_timestamp() + sqlc.arg(delay_seconds)::integer * INTERVAL '1 second'
WHERE name = sqlc.arg(name) AND fence = sqlc.arg(fence) AND lease_until > clock_timestamp();

-- name: TryWorkerTransactionLock :one
SELECT pg_try_advisory_xact_lock(hashtextextended(sqlc.arg(name)::text, 0))::boolean AS acquired;
