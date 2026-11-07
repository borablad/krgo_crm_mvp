-- name: CrmMetrics :one
select
  (select count(*) from clients)::bigint as clients_total,
  (select count(*) from deals where stage not in ('won', 'lost'))::bigint as deals_active,
  (select count(*) from deals where stage = 'won')::bigint as deals_won,
  (select coalesce(sum(amount), 0) from deals where stage not in ('won', 'lost'))::numeric(18,2) as deals_amount_active,
  (select coalesce(sum(amount), 0) from deals where stage = 'won')::numeric(18,2) as deals_amount_won,
  (select count(*) from tasks where status = 'open' and due_date < sqlc.arg('today')::date)::bigint as tasks_overdue,
  (select count(*) from tasks where status = 'open')::bigint as tasks_open;
