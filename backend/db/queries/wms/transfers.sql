-- name: NextTransferNumber :one
select 'ПМ-' || lpad(nextval('transfer_number_seq')::text, 6, '0') as number;

-- name: InsertTransfer :one
insert into transfers (number, warehouse_id, item_id, qty, from_cell_id, to_cell_id, comment, created_by)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id, created_at;

-- name: GetTransfer :one
select t.id, t.number, t.warehouse_id, w.name as warehouse_name, t.item_id, i.name as item_name, i.sku as item_sku,
       u.code as unit_code, t.qty, t.from_cell_id, fc.address as from_cell_address, t.to_cell_id, tc.address as to_cell_address,
       t.comment, t.created_by, cu.full_name as created_by_name, t.created_at
from transfers t
join warehouses w on w.id = t.warehouse_id
join items i on i.id = t.item_id
join units u on u.id = i.unit_id
join cells fc on fc.id = t.from_cell_id
join cells tc on tc.id = t.to_cell_id
join users cu on cu.id = t.created_by
where t.id = $1;

-- name: ListTransfers :many
select t.id, t.number, t.warehouse_id, w.name as warehouse_name, t.item_id, i.name as item_name, i.sku as item_sku,
       u.code as unit_code, t.qty, t.from_cell_id, fc.address as from_cell_address, t.to_cell_id, tc.address as to_cell_address,
       t.comment, t.created_by, cu.full_name as created_by_name, t.created_at
from transfers t
join warehouses w on w.id = t.warehouse_id
join items i on i.id = t.item_id
join units u on u.id = i.unit_id
join cells fc on fc.id = t.from_cell_id
join cells tc on tc.id = t.to_cell_id
join users cu on cu.id = t.created_by
where (sqlc.narg('warehouse_id')::bigint is null or t.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('item_id')::bigint is null or t.item_id = sqlc.narg('item_id'))
  and (sqlc.narg('cell_id')::bigint is null or t.from_cell_id = sqlc.narg('cell_id') or t.to_cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('date_from')::date is null or t.created_at >= sqlc.narg('date_from')::date::timestamp at time zone 'Asia/Almaty')
  and (sqlc.narg('date_to')::date is null or t.created_at < (sqlc.narg('date_to')::date + 1)::timestamp at time zone 'Asia/Almaty')
order by t.created_at desc, t.id desc
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountTransfers :one
select count(*)
from transfers t
where (sqlc.narg('warehouse_id')::bigint is null or t.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('item_id')::bigint is null or t.item_id = sqlc.narg('item_id'))
  and (sqlc.narg('cell_id')::bigint is null or t.from_cell_id = sqlc.narg('cell_id') or t.to_cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('date_from')::date is null or t.created_at >= sqlc.narg('date_from')::date::timestamp at time zone 'Asia/Almaty')
  and (sqlc.narg('date_to')::date is null or t.created_at < (sqlc.narg('date_to')::date + 1)::timestamp at time zone 'Asia/Almaty');
