-- name: GetAllSettings :many
SELECT key, value, updated_at FROM platform_settings ORDER BY key;

-- name: UpsertSetting :exec
INSERT INTO platform_settings (key, value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW();
