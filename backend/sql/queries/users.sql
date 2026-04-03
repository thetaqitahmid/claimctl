-- name: FindAllUsers :many
SELECT * FROM claimctl.users;

-- name: FindUserById :one
SELECT * FROM claimctl.users WHERE id=$1;

-- name: FindUserByEmail :one
SELECT * FROM claimctl.users WHERE email=$1;

-- name: CreateUser :exec
INSERT INTO claimctl.users (email, name, password, role, status, last_login) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetPasswordById :one
SELECT password FROM claimctl.users WHERE id=$1;

-- name: DeleteUserById :exec
DELETE FROM claimctl.users WHERE id=$1;

-- name: UpdateUserById :exec
UPDATE claimctl.users SET email=$1, name=$2, password=$3, role=$4, status=$5, last_login=$6, updated_at=CURRENT_TIMESTAMP
WHERE id=$7;

-- name: VerifyUserEmailIsUnique :one
SELECT COUNT(*) FROM claimctl.users where email=$1;

-- name: UpdateUserLastLogin :exec
UPDATE claimctl.users SET last_login=CURRENT_TIMESTAMP WHERE id=$1;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM claimctl.users WHERE role = 'admin';

-- name: UpdateUserFailedLoginAttempts :one
UPDATE claimctl.users
SET failed_login_attempts = failed_login_attempts + 1, locked_until = $1
WHERE id = $2
RETURNING failed_login_attempts;

-- name: ResetUserFailedLoginAttempts :exec
UPDATE claimctl.users
SET failed_login_attempts = 0, locked_until = NULL
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE claimctl.users
SET password = $1
WHERE id = $2;