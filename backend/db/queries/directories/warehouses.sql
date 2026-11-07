-- name: ListWarehouses :many
select id, name, code, responsible_user_id, is_active, created_at, updated_at
from warehouses
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(code) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountWarehouses :one
select count(*)
from warehouses
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(code) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetWarehouse :one
select id, name, code, responsible_user_id, is_active, created_at, updated_at
from warehouses
where id = $1;

-- name: CreateWarehouse :one
insert into warehouses (name, code, responsible_user_id)
values ($1, $2, $3)
returning id, name, code, responsible_user_id, is_active, created_at, updated_at;

-- name: UpdateWarehouse :one
update warehouses
set name = $2, code = $3, responsible_user_id = $4, is_active = $5, updated_at = now()
where id = $1
returning id, name, code, responsible_user_id, is_active, created_at, updated_at;

-- name: UserExists :one
select exists (select 1 from users where id = $1);

-- name: ListCells :many
select id, warehouse_id, address, is_receiving, is_active, created_at
from cells
where warehouse_id = sqlc.arg('warehouse_id')
  and (sqlc.narg('q')::text is null or lower(address) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by is_receiving desc, address, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountCells :one
select count(*)
from cells
where warehouse_id = sqlc.arg('warehouse_id')
  and (sqlc.narg('q')::text is null or lower(address) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetCell :one
select id, warehouse_id, address, is_receiving, is_active, created_at
from cells
where id = $1;

-- name: GetReceivingCell :one
select id, warehouse_id, address, is_receiving, is_active, created_at
from cells
where warehouse_id = $1 and is_receiving;

-- name: CreateCell :one
insert into cells (warehouse_id, address, is_receiving)
values ($1, $2, $3)
returning id, warehouse_id, address, is_receiving, is_active, created_at;

-- name: UpdateCell :one
update cells
set address = $2, is_active = $3
where id = $1
returning id, warehouse_id, address, is_receiving, is_active, created_at;

-- Варианты с блокировкой на чтение: используются внутри чужих транзакций (склад), чтобы
-- склад или ячейку не отключили между проверкой и записью.
-- name: GetWarehouseShare :one
select id, name, code, responsible_user_id, is_active, created_at, updated_at
from warehouses
where id = $1
for share;

-- name: GetCellShare :one
select id, warehouse_id, address, is_receiving, is_active, created_at
from cells
where id = $1
for share;

-- name: GetReceivingCellShare :one
select id, warehouse_id, address, is_receiving, is_active, created_at
from cells
where warehouse_id = $1 and is_receiving
for share;
