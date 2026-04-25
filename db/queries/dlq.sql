-- name: CreateDLQEntry :one
INSERT INTO dlq (
  id,
  tenant_id,
  endpoint_id,
  rule_id,
  destination_id,
  payload_b64,
  last_error,
  attempts,
  parked_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(endpoint_id),
  sqlc.narg(rule_id),
  sqlc.narg(destination_id),
  sqlc.arg(payload_b64),
  sqlc.narg(last_error),
  sqlc.arg(attempts),
  sqlc.arg(parked_at)
)
RETURNING *;

-- name: GetDLQEntryByTenant :one
SELECT *
FROM dlq
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);

-- name: ListDLQEntriesByTenant :many
SELECT *
FROM dlq
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY parked_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: ListDLQEntriesByEndpoint :many
SELECT *
FROM dlq
WHERE tenant_id = sqlc.arg(tenant_id)
  AND endpoint_id = sqlc.arg(endpoint_id)
ORDER BY parked_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: DeleteDLQEntry :exec
DELETE FROM dlq
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);

-- name: DeleteDLQEntriesBefore :execrows
DELETE FROM dlq
WHERE parked_at < sqlc.arg(before_ms);
