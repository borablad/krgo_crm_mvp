-- name: ListCounterparties :many
select id, name, bin, kind, contact_name, phone, email, is_active, created_at, updated_at
from counterparties
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or bin like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('kind')::text is null or kind = sqlc.narg('kind')::text)
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountCounterparties :one
select count(*)
from counterparties
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or bin like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('kind')::text is null or kind = sqlc.narg('kind')::text)
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetCounterparty :one
select id, name, bin, kind, contact_name, phone, email, is_active, created_at, updated_at
from counterparties
where id = $1;

-- name: CreateCounterparty :one
insert into counterparties (name, bin, kind, contact_name, phone, email)
values ($1, $2, $3, $4, $5, $6)
returning id, name, bin, kind, contact_name, phone, email, is_active, created_at, updated_at;

-- name: UpdateCounterparty :one
update counterparties
set name = $2, bin = $3, kind = $4, contact_name = $5, phone = $6, email = $7, is_active = $8, updated_at = now()
where id = $1
returning id, name, bin, kind, contact_name, phone, email, is_active, created_at, updated_at;
