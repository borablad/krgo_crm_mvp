-- name: ListUsers :many
select id, login, full_name, role, is_active, created_at
from users
order by full_name;

-- name: ListActiveUsers :many
select id, full_name, role
from users
where is_active
order by full_name;

-- name: GetUser :one
select id, login, full_name, role, is_active, created_at
from users
where id = $1;

-- name: CreateUser :one
insert into users (login, full_name, role, password_hash)
values ($1, $2, $3, $4)
returning id, login, full_name, role, is_active, created_at;

-- name: UpdateUser :one
update users
set full_name = $2, role = $3, is_active = $4, updated_at = now()
where id = $1
returning id, login, full_name, role, is_active, created_at;

-- name: SetUserPassword :exec
update users set password_hash = $2, updated_at = now() where id = $1;

-- name: CountActiveAdmins :one
select count(*) from users where role = 'admin' and is_active;
