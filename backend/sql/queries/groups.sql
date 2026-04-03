-- name: CreateGroup :one
INSERT INTO claimctl.groups (
  name, description
) VALUES (
  $1, $2
)
RETURNING *;

-- name: GetGroup :one
SELECT * FROM claimctl.groups
WHERE id = $1 LIMIT 1;

-- name: GetGroupByName :one
SELECT * FROM claimctl.groups
WHERE name = $1 LIMIT 1;

-- name: ListGroups :many
SELECT * FROM claimctl.groups
ORDER BY name;

-- name: UpdateGroup :one
UPDATE claimctl.groups
SET name = $2,
    description = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM claimctl.groups
WHERE id = $1;

-- name: AddUserToGroup :exec
INSERT INTO claimctl.group_members (
  group_id, user_id
) VALUES (
  $1, $2
)
ON CONFLICT DO NOTHING;

-- name: RemoveUserFromGroup :exec
DELETE FROM claimctl.group_members
WHERE group_id = $1 AND user_id = $2;

-- name: ListGroupMembers :many
SELECT u.id, u.email, u.name, u.role
FROM claimctl.users u
JOIN claimctl.group_members gm ON u.id = gm.user_id
WHERE gm.group_id = $1;

-- name: GetUserGroups :many
SELECT g.*
FROM claimctl.groups g
JOIN claimctl.group_members gm ON g.id = gm.group_id
WHERE gm.user_id = $1;

-- name: AddSpacePermission :exec
INSERT INTO claimctl.space_permissions (
    space_id, group_id, user_id
) VALUES (
    $1, $2, $3
)
ON CONFLICT DO NOTHING;

-- name: RemoveSpacePermission :exec
DELETE FROM claimctl.space_permissions
WHERE space_id = $1 AND (
    (group_id = $2) OR
    (user_id = $3)
);

-- name: GetSpacePermissions :many
SELECT sp.*,
       g.name as group_name,
       u.email as user_email
FROM claimctl.space_permissions sp
LEFT JOIN claimctl.groups g ON sp.group_id = g.id
LEFT JOIN claimctl.users u ON sp.user_id = u.id
WHERE sp.space_id = $1;
