-- name: ListClients :many
select c.id, c.name, c.bin, c.industry, c.region, c.contact_name, c.contact_position, c.phone, c.email,
       c.owner_user_id, u.full_name as owner_name, c.status, c.created_at, c.updated_at
from clients c
join users u on u.id = c.owner_user_id
where (sqlc.narg('q')::text is null
        or c.name ilike '%' || sqlc.narg('q') || '%'
        or c.bin ilike '%' || sqlc.narg('q') || '%'
        or c.contact_name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('owner_user_id')::bigint is null or c.owner_user_id = sqlc.narg('owner_user_id'))
  and (sqlc.narg('status')::text is null or c.status = sqlc.narg('status'))
order by lower(c.name), c.id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountClients :one
select count(*)
from clients c
where (sqlc.narg('q')::text is null
        or c.name ilike '%' || sqlc.narg('q') || '%'
        or c.bin ilike '%' || sqlc.narg('q') || '%'
        or c.contact_name ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('owner_user_id')::bigint is null or c.owner_user_id = sqlc.narg('owner_user_id'))
  and (sqlc.narg('status')::text is null or c.status = sqlc.narg('status'));

-- name: GetClient :one
select c.id, c.name, c.bin, c.industry, c.region, c.contact_name, c.contact_position, c.phone, c.email,
       c.owner_user_id, u.full_name as owner_name, c.status, c.created_at, c.updated_at
from clients c
join users u on u.id = c.owner_user_id
where c.id = $1;

-- name: CreateClient :one
insert into clients (name, bin, industry, region, contact_name, contact_position, phone, email, owner_user_id, status)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning id;

-- name: UpdateClient :one
update clients
set name = $2, bin = $3, industry = $4, region = $5, contact_name = $6, contact_position = $7,
    phone = $8, email = $9, owner_user_id = $10, status = $11, updated_at = now()
where id = $1
returning id;

-- name: ActiveUserExists :one
select exists (select 1 from users where id = $1 and is_active);

-- name: ClientExists :one
select exists (select 1 from clients where id = $1);

-- name: CountAllClients :one
select count(*) from clients;
