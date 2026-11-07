-- Блокировка строк остатка товара в обеих ячейках всегда в порядке cell_id: встречные перемещения не уходят в взаимную блокировку.
-- name: LockBalances :many
select cell_id, qty from stock_balances
where item_id = sqlc.arg('item_id') and cell_id = any(sqlc.arg('cell_ids')::bigint[])
order by cell_id
for update;

-- name: AddBalance :exec
insert into stock_balances (item_id, warehouse_id, cell_id, qty)
values ($1, $2, $3, $4)
on conflict (item_id, cell_id) do update set qty = stock_balances.qty + excluded.qty, updated_at = now();

-- name: SubBalance :exec
update stock_balances set qty = qty - sqlc.arg('qty'), updated_at = now()
where item_id = sqlc.arg('item_id') and cell_id = sqlc.arg('cell_id');

-- name: InsertMovement :exec
insert into stock_movements (warehouse_id, item_id, cell_id, qty_delta, document_type, document_id, user_id)
values ($1, $2, $3, $4, $5, $6, $7);

-- name: ListBalances :many
select b.item_id, i.name as item_name, i.sku as item_sku, u.code as unit_code,
       b.warehouse_id, w.name as warehouse_name, b.cell_id, c.address as cell_address, c.is_receiving, b.qty, b.updated_at
from stock_balances b
join items i on i.id = b.item_id
join units u on u.id = i.unit_id
join warehouses w on w.id = b.warehouse_id
join cells c on c.id = b.cell_id
where (sqlc.narg('q')::text is null
        or i.name ilike '%' || sqlc.narg('q') || '%'
        or i.sku ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('warehouse_id')::bigint is null or b.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('cell_id')::bigint is null or b.cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('item_id')::bigint is null or b.item_id = sqlc.narg('item_id'))
  and (sqlc.arg('include_zero')::boolean or b.qty > 0)
order by i.name, c.address, b.item_id, b.cell_id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountBalances :one
select count(*)
from stock_balances b
join items i on i.id = b.item_id
where (sqlc.narg('q')::text is null
        or i.name ilike '%' || sqlc.narg('q') || '%'
        or i.sku ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('warehouse_id')::bigint is null or b.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('cell_id')::bigint is null or b.cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('item_id')::bigint is null or b.item_id = sqlc.narg('item_id'))
  and (sqlc.arg('include_zero')::boolean or b.qty > 0);

-- name: ListMovements :many
select m.id, m.at, m.warehouse_id, w.name as warehouse_name, m.item_id, i.name as item_name, i.sku as item_sku,
       un.code as unit_code, m.cell_id, c.address as cell_address, m.qty_delta, m.document_type, m.document_id,
       coalesce(r.number, t.number)::text as document_number, m.user_id, u.full_name as user_name
from stock_movements m
join warehouses w on w.id = m.warehouse_id
join items i on i.id = m.item_id
join units un on un.id = i.unit_id
join cells c on c.id = m.cell_id
left join users u on u.id = m.user_id
left join receipts r on m.document_type = 'receipt' and r.id = m.document_id
left join transfers t on m.document_type = 'transfer' and t.id = m.document_id
where (sqlc.narg('item_id')::bigint is null or m.item_id = sqlc.narg('item_id'))
  and (sqlc.narg('warehouse_id')::bigint is null or m.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('cell_id')::bigint is null or m.cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('document_type')::text is null or m.document_type = sqlc.narg('document_type'))
  and (sqlc.narg('date_from')::date is null or m.at >= sqlc.narg('date_from')::date::timestamp at time zone 'Asia/Almaty')
  and (sqlc.narg('date_to')::date is null or m.at < (sqlc.narg('date_to')::date + 1)::timestamp at time zone 'Asia/Almaty')
order by m.at desc, m.id desc
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountMovements :one
select count(*)
from stock_movements m
where (sqlc.narg('item_id')::bigint is null or m.item_id = sqlc.narg('item_id'))
  and (sqlc.narg('warehouse_id')::bigint is null or m.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('cell_id')::bigint is null or m.cell_id = sqlc.narg('cell_id'))
  and (sqlc.narg('document_type')::text is null or m.document_type = sqlc.narg('document_type'))
  and (sqlc.narg('date_from')::date is null or m.at >= sqlc.narg('date_from')::date::timestamp at time zone 'Asia/Almaty')
  and (sqlc.narg('date_to')::date is null or m.at < (sqlc.narg('date_to')::date + 1)::timestamp at time zone 'Asia/Almaty');
