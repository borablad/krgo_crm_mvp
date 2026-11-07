-- name: ListOrganizations :many
select id, name, bin, address, is_active, created_at, updated_at
from organizations
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or bin like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean)
order by name, id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountOrganizations :one
select count(*)
from organizations
where (sqlc.narg('q')::text is null
       or lower(name) like lower('%' || sqlc.narg('q')::text || '%')
       or bin like '%' || sqlc.narg('q')::text || '%')
  and (sqlc.narg('is_active')::boolean is null or is_active = sqlc.narg('is_active')::boolean);

-- name: GetOrganization :one
select id, name, bin, address, is_active, created_at, updated_at
from organizations
where id = $1;

-- name: CreateOrganization :one
insert into organizations (name, bin, address)
values ($1, $2, $3)
returning id, name, bin, address, is_active, created_at, updated_at;

-- name: UpdateOrganization :one
update organizations
set name = $2, bin = $3, address = $4, is_active = $5, updated_at = now()
where id = $1
returning id, name, bin, address, is_active, created_at, updated_at;
