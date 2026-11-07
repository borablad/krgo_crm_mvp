-- name: GetUserByLogin :one
select id, login, full_name, role, password_hash, is_active
from users
where login = $1;

-- name: CreateSession :exec
insert into sessions (id, user_id, expires_at, user_agent, ip)
values ($1, $2, $3, $4, $5);

-- name: GetSessionUser :one
select s.id as session_id, s.expires_at,
       u.id, u.login, u.full_name, u.role, u.is_active
from sessions s
join users u on u.id = s.user_id
where s.id = $1;

-- name: TouchSession :exec
update sessions set last_seen_at = now() where id = $1;

-- name: DeleteSession :exec
delete from sessions where id = $1;

-- name: DeleteUserSessions :exec
delete from sessions where user_id = $1;

-- name: DeleteExpiredSessions :exec
delete from sessions where expires_at < now();
