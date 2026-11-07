-- name: NextRequestSeq :one
select nextval('payment_request_number_seq')::bigint;

-- name: InsertRequest :one
insert into payment_requests (number, organization_id, counterparty_id, basis, amount, currency, pay_date,
                              cashflow_item_id, cost_center_id, comment, created_by)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
returning id;

-- name: UpdateRequest :exec
update payment_requests
set organization_id = $2, counterparty_id = $3, basis = $4, amount = $5, currency = $6, pay_date = $7,
    cashflow_item_id = $8, cost_center_id = $9, comment = $10, updated_at = now()
where id = $1;

-- name: GetRequestForUpdate :one
select id, number, status, created_by from payment_requests where id = $1 for update;

-- name: SetRequestStatus :exec
update payment_requests
set status = sqlc.arg('status'), decided_by = coalesce(sqlc.narg('decided_by')::bigint, decided_by),
    decided_at = coalesce(sqlc.narg('decided_at')::timestamptz, decided_at),
    paid_at = coalesce(sqlc.narg('paid_at')::timestamptz, paid_at), updated_at = now()
where id = sqlc.arg('id');

-- name: InsertRequestHistory :exec
insert into payment_request_history (request_id, from_status, to_status, user_id, comment)
values ($1, $2, $3, $4, $5);

-- name: ListRequestHistory :many
select h.id, h.request_id, h.from_status, h.to_status, h.user_id, u.full_name as user_name, h.at, h.comment
from payment_request_history h
left join users u on u.id = h.user_id
where h.request_id = $1
order by h.at, h.id;

-- name: GetRequest :one
select r.id, r.number, r.organization_id, o.name as organization_name, r.counterparty_id, c.name as counterparty_name,
       r.basis, r.amount, r.currency, r.pay_date, r.cashflow_item_id, ci.name as cashflow_item_name,
       r.cost_center_id, cc.name as cost_center_name, r.comment, r.status,
       r.created_by, cu.full_name as created_by_name, r.decided_by, du.full_name as decided_by_name,
       r.decided_at, r.paid_at, r.created_at, r.updated_at
from payment_requests r
join organizations o on o.id = r.organization_id
join counterparties c on c.id = r.counterparty_id
join cashflow_items ci on ci.id = r.cashflow_item_id
join cost_centers cc on cc.id = r.cost_center_id
join users cu on cu.id = r.created_by
left join users du on du.id = r.decided_by
where r.id = $1;

-- name: ListRequests :many
select r.id, r.number, r.organization_id, o.name as organization_name, r.counterparty_id, c.name as counterparty_name,
       r.basis, r.amount, r.currency, r.pay_date, r.cashflow_item_id, ci.name as cashflow_item_name,
       r.cost_center_id, cc.name as cost_center_name, r.comment, r.status,
       r.created_by, cu.full_name as created_by_name, r.decided_by, du.full_name as decided_by_name,
       r.decided_at, r.paid_at, r.created_at, r.updated_at
from payment_requests r
join organizations o on o.id = r.organization_id
join counterparties c on c.id = r.counterparty_id
join cashflow_items ci on ci.id = r.cashflow_item_id
join cost_centers cc on cc.id = r.cost_center_id
join users cu on cu.id = r.created_by
left join users du on du.id = r.decided_by
where (sqlc.narg('q')::text is null or r.number ilike '%' || sqlc.narg('q') || '%'
       or r.basis ilike '%' || sqlc.narg('q') || '%' or c.name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('status')::text is null or r.status = sqlc.narg('status'))
  and (sqlc.narg('organization_id')::bigint is null or r.organization_id = sqlc.narg('organization_id'))
  and (sqlc.narg('counterparty_id')::bigint is null or r.counterparty_id = sqlc.narg('counterparty_id'))
  and (sqlc.narg('currency')::text is null or r.currency = sqlc.narg('currency'))
  and (sqlc.narg('pay_date_from')::date is null or r.pay_date >= sqlc.narg('pay_date_from'))
  and (sqlc.narg('pay_date_to')::date is null or r.pay_date <= sqlc.narg('pay_date_to'))
  and (sqlc.narg('created_by')::bigint is null or r.created_by = sqlc.narg('created_by'))
order by r.pay_date, r.id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountRequests :one
select count(*)
from payment_requests r
join counterparties c on c.id = r.counterparty_id
where (sqlc.narg('q')::text is null or r.number ilike '%' || sqlc.narg('q') || '%'
       or r.basis ilike '%' || sqlc.narg('q') || '%' or c.name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('status')::text is null or r.status = sqlc.narg('status'))
  and (sqlc.narg('organization_id')::bigint is null or r.organization_id = sqlc.narg('organization_id'))
  and (sqlc.narg('counterparty_id')::bigint is null or r.counterparty_id = sqlc.narg('counterparty_id'))
  and (sqlc.narg('currency')::text is null or r.currency = sqlc.narg('currency'))
  and (sqlc.narg('pay_date_from')::date is null or r.pay_date >= sqlc.narg('pay_date_from'))
  and (sqlc.narg('pay_date_to')::date is null or r.pay_date <= sqlc.narg('pay_date_to'))
  and (sqlc.narg('created_by')::bigint is null or r.created_by = sqlc.narg('created_by'));

-- name: CountAllRequests :one
select count(*) from payment_requests;
