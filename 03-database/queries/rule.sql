-- name: CreateRule :one
INSERT INTO rule (
  id,
  endpoint_id,
  position,
  name,
  filter_jsonlogic,
  transform_kind,
  transform_body,
  destination_id,
  on_match,
  enabled,
  created_at
)
SELECT
  sqlc.arg(id),
  e.id,
  sqlc.arg(position),
  sqlc.arg(name),
  sqlc.narg(filter_jsonlogic),
  sqlc.arg(transform_kind),
  sqlc.narg(transform_body),
  sqlc.narg(destination_id),
  sqlc.arg(on_match),
  sqlc.arg(enabled),
  sqlc.arg(created_at)
FROM endpoint e
WHERE e.tenant_id = sqlc.arg(tenant_id)
  AND e.id = sqlc.arg(endpoint_id)
RETURNING *;

-- name: GetRuleByTenant :one
SELECT r.*
FROM rule r
JOIN endpoint e ON e.id = r.endpoint_id
WHERE e.tenant_id = sqlc.arg(tenant_id)
  AND r.id = sqlc.arg(id);

-- name: ListRulesByEndpoint :many
SELECT r.*
FROM rule r
JOIN endpoint e ON e.id = r.endpoint_id
WHERE e.tenant_id = sqlc.arg(tenant_id)
  AND r.endpoint_id = sqlc.arg(endpoint_id)
ORDER BY r.position ASC;

-- name: ListEnabledRulesForEndpoint :many
SELECT *
FROM rule
WHERE endpoint_id = sqlc.arg(endpoint_id)
  AND enabled = 1
ORDER BY position ASC;

-- name: UpdateRule :one
UPDATE rule
SET position = sqlc.arg(position),
    name = sqlc.arg(name),
    filter_jsonlogic = sqlc.narg(filter_jsonlogic),
    transform_kind = sqlc.arg(transform_kind),
    transform_body = sqlc.narg(transform_body),
    destination_id = sqlc.narg(destination_id),
    on_match = sqlc.arg(on_match),
    enabled = sqlc.arg(enabled)
WHERE id = sqlc.arg(id)
  AND endpoint_id IN (
    SELECT id
    FROM endpoint
    WHERE tenant_id = sqlc.arg(tenant_id)
  )
RETURNING *;

-- name: DeleteRule :exec
DELETE FROM rule
WHERE id = sqlc.arg(id)
  AND endpoint_id IN (
    SELECT id
    FROM endpoint
    WHERE tenant_id = sqlc.arg(tenant_id)
  );
