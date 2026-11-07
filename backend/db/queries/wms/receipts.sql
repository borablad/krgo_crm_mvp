-- name: NextReceiptNumber :one
select ('ПС-' || (lpad(nextval('receipt_number_seq')::text, 6, '0'))::text)::text as number;

-- name: InsertReceipt :one
insert into receipts (number, warehouse_id, counterparty_id, doc_date, comment, created_by)
values ($1, $2, $3, $4, $5, $6)
returning id;

-- name: UpdateReceiptHeader :exec
update receipts
set warehouse_id = $2, counterparty_id = $3, doc_date = $4, comment = $5, updated_at = now()
where id = $1;

-- name: GetReceiptForUpdate :one
select id, number, status, warehouse_id, doc_date from receipts where id = $1 for update;

-- name: MarkReceiptPosted :exec
update receipts set status = 'posted', posted_by = $2, posted_at = now(), updated_at = now() where id = $1;

-- name: GetReceipt :one
select r.id, r.number, r.warehouse_id, w.name as warehouse_name, r.counterparty_id, c.name as counterparty_name,
       r.doc_date, r.comment, r.status, r.created_by, cu.full_name as created_by_name,
       r.posted_by, pu.full_name as posted_by_name, r.posted_at, r.created_at, r.updated_at
from receipts r
join warehouses w on w.id = r.warehouse_id
join counterparties c on c.id = r.counterparty_id
join users cu on cu.id = r.created_by
left join users pu on pu.id = r.posted_by
where r.id = $1;

-- name: ListReceipts :many
select r.id, r.number, r.warehouse_id, w.name as warehouse_name, r.counterparty_id, c.name as counterparty_name,
       r.doc_date, r.comment, r.status, r.created_by, cu.full_name as created_by_name,
       r.posted_by, pu.full_name as posted_by_name, r.posted_at, r.created_at, r.updated_at
from receipts r
join warehouses w on w.id = r.warehouse_id
join counterparties c on c.id = r.counterparty_id
join users cu on cu.id = r.created_by
left join users pu on pu.id = r.posted_by
where (sqlc.narg('q')::text is null
        or r.number ilike '%' || sqlc.narg('q') || '%'
        or c.name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('warehouse_id')::bigint is null or r.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('status')::text is null or r.status = sqlc.narg('status'))
  and (sqlc.narg('date_from')::date is null or r.doc_date >= sqlc.narg('date_from'))
  and (sqlc.narg('date_to')::date is null or r.doc_date <= sqlc.narg('date_to'))
order by r.doc_date desc, r.id desc
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountReceipts :one
select count(*)
from receipts r
join counterparties c on c.id = r.counterparty_id
where (sqlc.narg('q')::text is null
        or r.number ilike '%' || sqlc.narg('q') || '%'
        or c.name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('warehouse_id')::bigint is null or r.warehouse_id = sqlc.narg('warehouse_id'))
  and (sqlc.narg('status')::text is null or r.status = sqlc.narg('status'))
  and (sqlc.narg('date_from')::date is null or r.doc_date >= sqlc.narg('date_from'))
  and (sqlc.narg('date_to')::date is null or r.doc_date <= sqlc.narg('date_to'));

-- name: DeleteReceiptLines :exec
delete from receipt_lines where receipt_id = $1;

-- name: InsertReceiptLine :exec
insert into receipt_lines (receipt_id, item_id, qty) values ($1, $2, $3);

-- name: ListReceiptLines :many
select l.id, l.receipt_id, l.item_id, i.name as item_name, i.sku as item_sku, u.code as unit_code, l.qty
from receipt_lines l
join items i on i.id = l.item_id
join units u on u.id = i.unit_id
where l.receipt_id = $1
order by l.id;

-- name: ReceiptLinesCount :many
select receipt_id, count(*)::bigint as lines_count
from receipt_lines
where receipt_id = any(sqlc.arg('ids')::bigint[])
group by receipt_id;
