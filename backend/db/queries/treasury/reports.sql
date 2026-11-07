-- name: CalendarTotals :many
-- Черновики и отклонённые в календарь не попадают.
select r.pay_date, r.currency,
       coalesce(sum(r.amount) filter (where r.status in ('pending', 'approved')), 0)::numeric(18,2) as planned_total,
       coalesce(sum(r.amount) filter (where r.status = 'approved'), 0)::numeric(18,2) as approved_total,
       coalesce(sum(r.amount) filter (where r.status = 'paid'), 0)::numeric(18,2) as paid_total,
       count(*) as cnt
from payment_requests r
where r.pay_date between sqlc.arg('date_from')::date and sqlc.arg('date_to')::date
  and r.status in ('pending', 'approved', 'paid')
  and (sqlc.narg('organization_id')::bigint is null or r.organization_id = sqlc.narg('organization_id'))
  and (sqlc.narg('currency')::text is null or r.currency = sqlc.narg('currency'))
  and (sqlc.narg('created_by')::bigint is null or r.created_by = sqlc.narg('created_by'))
group by r.pay_date, r.currency
order by r.pay_date, r.currency;

-- name: CalendarRequests :many
select r.id, r.number, r.pay_date, c.name as counterparty_name, r.amount, r.currency, r.status
from payment_requests r
join counterparties c on c.id = r.counterparty_id
where r.pay_date between sqlc.arg('date_from')::date and sqlc.arg('date_to')::date
  and r.status in ('pending', 'approved', 'paid')
  and (sqlc.narg('organization_id')::bigint is null or r.organization_id = sqlc.narg('organization_id'))
  and (sqlc.narg('currency')::text is null or r.currency = sqlc.narg('currency'))
  and (sqlc.narg('created_by')::bigint is null or r.created_by = sqlc.narg('created_by'))
order by r.pay_date, r.id;

-- name: CashflowReport :many
-- Плановый отчёт: ожидающие, согласованные и оплаченные заявки по статье, месяцу даты оплаты и валюте.
select r.cashflow_item_id, ci.name as cashflow_item_name,
       date_trunc('month', r.pay_date)::date as month, r.currency,
       coalesce(sum(r.amount) filter (where r.status = 'pending'), 0)::numeric(18,2) as pending_total,
       coalesce(sum(r.amount) filter (where r.status = 'approved'), 0)::numeric(18,2) as approved_total,
       coalesce(sum(r.amount) filter (where r.status = 'paid'), 0)::numeric(18,2) as paid_total,
       coalesce(sum(r.amount), 0)::numeric(18,2) as total,
       count(*) as cnt
from payment_requests r
join cashflow_items ci on ci.id = r.cashflow_item_id
where r.pay_date between sqlc.arg('date_from')::date and sqlc.arg('date_to')::date
  and r.status in ('pending', 'approved', 'paid')
  and (sqlc.narg('organization_id')::bigint is null or r.organization_id = sqlc.narg('organization_id'))
  and (sqlc.narg('created_by')::bigint is null or r.created_by = sqlc.narg('created_by'))
group by r.cashflow_item_id, ci.name, date_trunc('month', r.pay_date), r.currency
order by ci.name, month, r.currency;
