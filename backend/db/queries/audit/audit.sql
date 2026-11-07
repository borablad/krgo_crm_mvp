-- name: InsertAudit :exec
insert into audit_log (user_id, action, entity, entity_id, details)
values ($1, $2, $3, $4, $5);

-- name: ListAudit :many
select a.id, a.at, a.user_id, u.full_name as user_name, a.action, a.entity, a.entity_id, a.details
from audit_log a
left join users u on u.id = a.user_id
where (sqlc.narg('entity')::text is null or a.entity = sqlc.narg('entity')::text)
  and (sqlc.narg('user_id')::bigint is null or a.user_id = sqlc.narg('user_id')::bigint)
order by a.at desc, a.id desc
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountAudit :one
select count(*)
from audit_log a
where (sqlc.narg('entity')::text is null or a.entity = sqlc.narg('entity')::text)
  and (sqlc.narg('user_id')::bigint is null or a.user_id = sqlc.narg('user_id')::bigint);
