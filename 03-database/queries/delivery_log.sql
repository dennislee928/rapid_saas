-- name: CreateDeliveryLog :one
INSERT INTO delivery_log (
  id,
  tenant_id,
  endpoint_id,
  rule_id,
  destination_id,
  status,
  attempt,
  http_status,
  latency_ms,
  error,
  request_hash,
  request_size,
  received_at,
  delivered_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(endpoint_id),
  sqlc.narg(rule_id),
  sqlc.narg(destination_id),
  sqlc.arg(status),
  sqlc.arg(attempt),
  sqlc.narg(http_status),
  sqlc.narg(latency_ms),
  sqlc.narg(error),
  sqlc.narg(request_hash),
  sqlc.narg(request_size),
  sqlc.arg(received_at),
  sqlc.narg(delivered_at)
)
RETURNING *;

-- name: GetDeliveryLogByTenant :one
SELECT *
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(id);

-- name: ListDeliveryLogsByTenant :many
SELECT *
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND received_at >= sqlc.arg(since_ms)
ORDER BY received_at DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListDeliveryLogsByEndpoint :many
SELECT *
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND endpoint_id = sqlc.arg(endpoint_id)
  AND received_at >= sqlc.arg(since_ms)
ORDER BY received_at DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListDeliveryLogsByStatus :many
SELECT *
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND status = sqlc.arg(status)
  AND received_at >= sqlc.arg(since_ms)
ORDER BY received_at DESC
LIMIT sqlc.arg(limit_rows);

-- name: FindDeliveryLogsByRequestHash :many
SELECT *
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND request_hash = sqlc.arg(request_hash)
  AND received_at >= sqlc.arg(since_ms)
ORDER BY received_at DESC
LIMIT sqlc.arg(limit_rows);

-- name: CountDeliveryLogsByStatus :many
SELECT status, COUNT(*) AS count
FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND received_at >= sqlc.arg(since_ms)
GROUP BY status;

-- name: DeleteDeliveryLogsBefore :execrows
DELETE FROM delivery_log
WHERE received_at < sqlc.arg(before_ms);

-- name: DeleteTenantDeliveryLogsBefore :execrows
DELETE FROM delivery_log
WHERE tenant_id = sqlc.arg(tenant_id)
  AND received_at < sqlc.arg(before_ms);
