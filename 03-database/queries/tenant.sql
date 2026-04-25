-- name: CreateTenant :one
INSERT INTO tenant (id, name, plan, stripe_cust_id, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(plan), sqlc.narg(stripe_cust_id), sqlc.arg(created_at))
RETURNING *;

-- name: GetTenant :one
SELECT *
FROM tenant
WHERE id = sqlc.arg(id);

-- name: ListTenants :many
SELECT *
FROM tenant
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: UpdateTenantPlan :one
UPDATE tenant
SET plan = sqlc.arg(plan),
    stripe_cust_id = sqlc.narg(stripe_cust_id)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteTenant :exec
DELETE FROM tenant
WHERE id = sqlc.arg(id);
