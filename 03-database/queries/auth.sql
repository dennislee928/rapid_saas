-- name: CreateUser :one
INSERT INTO user (id, tenant_id, email, clerk_user_id, role, created_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(email),
  sqlc.narg(clerk_user_id),
  sqlc.arg(role),
  sqlc.arg(created_at)
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM user
WHERE email = sqlc.arg(email);

-- name: ListUsersByTenant :many
SELECT *
FROM user
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY created_at ASC;

-- name: CreateAPIKey :one
INSERT INTO api_key (id, tenant_id, prefix, hash, scopes, last_used_at, revoked_at, created_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(prefix),
  sqlc.arg(hash),
  sqlc.arg(scopes),
  sqlc.narg(last_used_at),
  sqlc.narg(revoked_at),
  sqlc.arg(created_at)
)
RETURNING *;

-- name: GetAPIKeyByPrefix :one
SELECT *
FROM api_key
WHERE prefix = sqlc.arg(prefix)
  AND revoked_at IS NULL;

-- name: ListAPIKeysByTenant :many
SELECT *
FROM api_key
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY created_at DESC;

-- name: MarkAPIKeyUsed :exec
UPDATE api_key
SET last_used_at = sqlc.arg(last_used_at)
WHERE id = sqlc.arg(id);

-- name: RevokeAPIKey :one
UPDATE api_key
SET revoked_at = sqlc.arg(revoked_at)
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id)
RETURNING *;
