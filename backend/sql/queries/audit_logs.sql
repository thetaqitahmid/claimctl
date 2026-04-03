-- name: CreateAuditLog :one
INSERT INTO claimctl.audit_logs (
    actor_id,
    action,
    entity_type,
    entity_id,
    changes,
    ip_address,
    created_at
) VALUES (
             $1, $2, $3, $4, $5, $6, EXTRACT(EPOCH FROM NOW())::BIGINT
         )
RETURNING *;

-- name: GetAuditLogs :many
SELECT a.*, u.email as actor_email
FROM claimctl.audit_logs a
LEFT JOIN claimctl.users u ON a.actor_id = u.id
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;
