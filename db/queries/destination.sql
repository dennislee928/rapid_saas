-- name: CreateDestination :one
INSERT INTO destination (id, tenant_id, kind, name, config_json, secret_ref, created_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(kind),
  sqlc.arg(name),
  sqlc.arg(config_json),
  sqlc.narg(secret_ref),
  sqlc.arg(created_at)
)
RETURNING *;

-- name: GetDestinationByTenant :one
SELECT *
FROM destination
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);

-- name: ListDestinationsByTenant :many
SELECT *
FROM destination
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: UpdateDestination :one
UPDATE destination
SET kind = sqlc.arg(kind),
    name = sqlc.arg(name),
    config_json = sqlc.arg(config_json),
    secret_ref = sqlc.narg(secret_ref)
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteDestination :exec
DELETE FROM destination
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);
