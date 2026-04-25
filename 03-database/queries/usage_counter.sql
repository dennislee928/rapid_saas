-- name: UpsertUsageCounter :one
INSERT INTO usage_counter (tenant_id, bucket_hour, ingressed, forwarded, failed)
VALUES (
  sqlc.arg(tenant_id),
  sqlc.arg(bucket_hour),
  sqlc.arg(ingressed),
  sqlc.arg(forwarded),
  sqlc.arg(failed)
)
ON CONFLICT (tenant_id, bucket_hour) DO UPDATE SET
  ingressed = usage_counter.ingressed + excluded.ingressed,
  forwarded = usage_counter.forwarded + excluded.forwarded,
  failed = usage_counter.failed + excluded.failed
RETURNING *;

-- name: GetUsageCounterBucket :one
SELECT *
FROM usage_counter
WHERE tenant_id = sqlc.arg(tenant_id)
  AND bucket_hour = sqlc.arg(bucket_hour);

-- name: ListUsageCounters :many
SELECT *
FROM usage_counter
WHERE tenant_id = sqlc.arg(tenant_id)
  AND bucket_hour BETWEEN sqlc.arg(from_bucket_hour) AND sqlc.arg(to_bucket_hour)
ORDER BY bucket_hour ASC;

-- name: SumUsageCounters :one
SELECT
  COALESCE(SUM(ingressed), 0) AS ingressed,
  COALESCE(SUM(forwarded), 0) AS forwarded,
  COALESCE(SUM(failed), 0) AS failed
FROM usage_counter
WHERE tenant_id = sqlc.arg(tenant_id)
  AND bucket_hour BETWEEN sqlc.arg(from_bucket_hour) AND sqlc.arg(to_bucket_hour);

-- name: DeleteUsageCountersBefore :execrows
DELETE FROM usage_counter
WHERE bucket_hour < sqlc.arg(before_bucket_hour);
