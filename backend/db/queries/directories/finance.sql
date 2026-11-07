-- name: ListCashflowItems :many
select id, name, kind, is_active, created_at, updated_at
from cashflow_items
where (sqlc.narg('q')::text is null or lower(name) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('kind')::text is null or kind = sqlc.narg('kind')::text)
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountCashflowItems :one
select count(*)
from cashflow_items
where (sqlc.narg('q')::text is null or lower(name) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('kind')::text is null or kind = sqlc.narg('kind')::text)
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetCashflowItem :one
select id, name, kind, is_active, created_at, updated_at
from cashflow_items
where id = $1;

-- name: CreateCashflowItem :one
insert into cashflow_items (name, kind)
values ($1, $2)
returning id, name, kind, is_active, created_at, updated_at;

-- name: UpdateCashflowItem :one
update cashflow_items
set name = $2, kind = $3, is_active = $4, updated_at = now()
where id = $1
returning id, name, kind, is_active, created_at, updated_at;

-- name: ListCostCenters :many
select id, name, code, is_active, created_at, updated_at
from cost_centers
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(code) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountCostCenters :one
select count(*)
from cost_centers
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or lower(code) like lower('%' || sqlc.narg('q')::text || '%'))
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetCostCenter :one
select id, name, code, is_active, created_at, updated_at
from cost_centers
where id = $1;

-- name: CreateCostCenter :one
insert into cost_centers (name, code)
values ($1, $2)
returning id, name, code, is_active, created_at, updated_at;

-- name: UpdateCostCenter :one
update cost_centers
set name = $2, code = $3, is_active = $4, updated_at = now()
where id = $1
returning id, name, code, is_active, created_at, updated_at;

-- name: ListCurrencies :many
select code, name, is_active from currencies order by code;

-- name: CurrencyExists :one
select exists (select 1 from currencies where code = $1 and is_active);
