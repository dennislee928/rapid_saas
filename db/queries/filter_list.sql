-- name: CreateFilterList :one
INSERT INTO filter_list (id, tenant_id, name, kind, source, refreshed_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(tenant_id),
  sqlc.arg(name),
  sqlc.arg(kind),
  sqlc.arg(source),
  sqlc.narg(refreshed_at)
)
RETURNING *;

-- name: ListFilterListsByTenant :many
SELECT *
FROM filter_list
WHERE tenant_id = sqlc.arg(tenant_id)
ORDER BY name ASC;

-- name: ReplaceFilterListItem :exec
INSERT OR REPLACE INTO filter_list_item (list_id, value)
VALUES (sqlc.arg(list_id), sqlc.arg(value));

-- name: ListFilterListItems :many
SELECT fli.value
FROM filter_list_item fli
JOIN filter_list fl ON fl.id = fli.list_id
WHERE fl.tenant_id = sqlc.arg(tenant_id)
  AND fli.list_id = sqlc.arg(list_id)
ORDER BY fli.value ASC;

-- name: DeleteFilterListItem :exec
DELETE FROM filter_list_item
WHERE list_id IN (
    SELECT id
    FROM filter_list
    WHERE tenant_id = sqlc.arg(tenant_id)
  )
  AND list_id = sqlc.arg(list_id)
  AND value = sqlc.arg(value);
