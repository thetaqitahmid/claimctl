-- name: GetSetting :one
SELECT * FROM app_settings
WHERE key = $1 LIMIT 1;

-- name: GetSettings :many
SELECT * FROM app_settings
ORDER BY category, key;

-- name: UpsertSetting :one
INSERT INTO app_settings (key, value, category, description, is_secret, updated_at)
VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    is_secret = EXCLUDED.is_secret,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;
