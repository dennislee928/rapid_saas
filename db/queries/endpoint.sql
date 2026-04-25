-- name: CreateEndpoint :one
INSERT INTO endpoint (
  id,
  tenant_id,
  name,
  source_preset,
  signing_secret,
  signing_header,
  signing_algo,
  enabled,
  created_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(name),
  sqlc.narg(source_preset),
  sqlc.narg(signing_secret),
  sqlc.narg(signing_header),
  sqlc.narg(signing_algo),
  sqlc.arg(enabled),
  sqlc.arg(created_at)
)
RETURNING *;

-- name: GetEndpointByTenant :one
SELECT *
FROM endpoint
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);

-- name: GetEndpointForIngress :one
SELECT e.*, t.plan
FROM endpoint e
JOIN tenant t ON t.id = e.tenant_id
WHERE e.id = sqlc.arg(id)
  AND e.enabled = 1;

-- name: ListEndpointsByTenant :many
SELECT *
FROM endpoint
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: UpdateEndpoint :one
UPDATE endpoint
SET name = sqlc.arg(name),
    source_preset = sqlc.narg(source_preset),
    signing_secret = sqlc.narg(signing_secret),
    signing_header = sqlc.narg(signing_header),
    signing_algo = sqlc.narg(signing_algo),
    enabled = sqlc.arg(enabled)
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: SetEndpointEnabled :one
UPDATE endpoint
SET enabled = sqlc.arg(enabled)
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteEndpoint :exec
DELETE FROM endpoint
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);
