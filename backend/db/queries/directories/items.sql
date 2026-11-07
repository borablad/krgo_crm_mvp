-- name: ListUnits :many
select id, code, name from units order by id;

-- name: UnitExists :one
select exists (select 1 from units where id = $1);

-- name: ListItems :many
select id, name, sku, unit_id, barcode, is_active, created_at, updated_at
from items
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(sku) like lower('%' || sqlc.narg('q')::text || '%')
       or barcode like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountItems :one
select count(*)
from items
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(sku) like lower('%' || sqlc.narg('q')::text || '%')
       or barcode like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetItem :one
select id, name, sku, unit_id, barcode, is_active, created_at, updated_at
from items
where id = $1;

-- name: CreateItem :one
insert into items (name, sku, unit_id, barcode)
values ($1, $2, $3, $4)
returning id, name, sku, unit_id, barcode, is_active, created_at, updated_at;

-- name: UpdateItem :one
update items
set name = $2, sku = $3, unit_id = $4, barcode = $5, is_active = $6, updated_at = now()
where id = $1
returning id, name, sku, unit_id, barcode, is_active, created_at, updated_at;
