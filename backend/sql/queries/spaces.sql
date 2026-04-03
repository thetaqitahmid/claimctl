-- name: CreateSpace :one
INSERT INTO claimctl.spaces (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetSpace :one
SELECT * FROM claimctl.spaces WHERE id = $1;

-- name: GetSpaceByName :one
SELECT * FROM claimctl.spaces WHERE name = $1;

-- name: ListSpaces :many
SELECT * FROM claimctl.spaces ORDER BY id;

-- name: ListSpacesForUser :many
SELECT DISTINCT s.*
FROM claimctl.spaces s
LEFT JOIN claimctl.space_permissions sp ON s.id = sp.space_id
LEFT JOIN claimctl.group_members gm ON sp.group_id = gm.group_id
WHERE
    s.name = 'Default Space' OR
    sp.user_id = $1 OR
    (sp.group_id IS NOT NULL AND gm.user_id = $1)
ORDER BY s.id;

-- name: UpdateSpace :one
UPDATE claimctl.spaces
SET name = $1, description = $2, updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
WHERE id = $3
RETURNING *;

-- name: DeleteSpace :exec
DELETE FROM claimctl.spaces WHERE id = $1;

-- name: HasSpacePermission :one
SELECT EXISTS (
    SELECT 1 FROM claimctl.spaces s
    LEFT JOIN claimctl.space_permissions sp ON s.id = sp.space_id
    LEFT JOIN claimctl.group_members gm ON sp.group_id = gm.group_id
    WHERE s.id = $1 AND (
        s.name = 'Default Space' OR
        sp.user_id = $2 OR
        (sp.group_id IS NOT NULL AND gm.user_id = $2)
    )
);
